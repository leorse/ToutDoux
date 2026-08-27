package onnx

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"

	ort "github.com/yalue/onnxruntime_go"

	"toutdoux/adapters/model"
	"toutdoux/domain"
)

// Noms des entrées et de la sortie du graphe, lus dans le fichier `model.onnx`
// lui-même. Ils font partie du contrat du modèle et ne sont pas configurables.
//
// À noter : cet export prend bien `token_type_ids`, ce qu'on n'attendrait pas
// d'un XLM-RoBERTa. Le graphe le consomme, il faut donc le fournir — rempli de
// zéros, puisqu'il n'y a qu'un seul segment.
var (
	entrees = []string{"input_ids", "attention_mask", "token_type_ids"}
	sortie  = []string{"last_hidden_state"}
)

// Préfixes d'entraînement de la famille e5.
//
// Le modèle a appris à distinguer une requête d'un passage indexé par ces
// préfixes. Les omettre, ou les intervertir, dégrade nettement la pertinence :
// ce ne sont pas des décorations.
const (
	prefixeRequete = "query: "
	prefixePassage = "passage: "
)

// NomRuntime est le nom de la bibliothèque ONNX Runtime attendue.
const NomRuntime = "onnxruntime.dll"

// Provider vectorise avec le vrai modèle (§2.12, étape 5.7).
//
// **Chargement paresseux.** Ni le modèle ni le runtime ne sont touchés avant la
// première vectorisation réelle : charger 120 Mo au démarrage ferait sauter le
// budget de 500 ms du §3.8, et l'immense majorité des lancements n'utilisera
// jamais la recherche sémantique.
type Provider struct {
	// mu sérialise les vectorisations. Une seule inférence à la fois : c'est
	// déjà le schéma d'usage réel, et cela évite d'avoir à raisonner sur la
	// réentrance de la session ONNX.
	mu sync.Mutex

	dir     string // dossier des modèles (§3.3)
	runtime string // chemin de onnxruntime.dll, vide si introuvable

	tok     *tokenizer
	session *ort.DynamicAdvancedSession

	// echec retient une panne définitive de chargement. Sans lui, chaque
	// vectorisation retenterait de lire 120 Mo pour échouer pareil.
	echec error
}

// NewProvider construit le fournisseur pour un dossier de modèles.
//
// Il ne lit rien : la construction est gratuite et se fait au démarrage.
func NewProvider(modelDir string) *Provider {
	return &Provider{dir: modelDir, runtime: TrouverRuntime(modelDir)}
}

// TrouverRuntime cherche onnxruntime.dll et rend son chemin, ou "".
//
// Trois emplacements, dans cet ordre :
//
//  1. à côté de l'exécutable — c'est la distribution portable du §6.1, l'archive
//     contient `toutdoux.exe` et `onnxruntime.dll` côte à côte ;
//  2. le répertoire courant — c'est ce qui rend `wails dev` utilisable, où
//     l'exécutable est reconstruit dans un dossier temporaire ;
//  3. le dossier des modèles — dernier recours, mais c'est le seul dossier dont
//     l'utilisateur connaît déjà le chemin, et il est inscriptible.
func TrouverRuntime(modelDir string) string {
	var candidats []string
	if exe, err := os.Executable(); err == nil {
		candidats = append(candidats, filepath.Join(filepath.Dir(exe), NomRuntime))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidats = append(candidats, filepath.Join(cwd, NomRuntime))
	}
	candidats = append(candidats, filepath.Join(modelDir, NomRuntime))

	for _, c := range candidats {
		if info, err := os.Stat(c); err == nil && !info.IsDir() && info.Size() > 0 {
			return c
		}
	}
	return ""
}

// RuntimePath rend le chemin retenu pour la bibliothèque, ou "" si absente.
func (p *Provider) RuntimePath() string { return p.runtime }

// Available indique si une vectorisation est possible.
//
// Elle relit le disque mais **ne charge rien** : c'est ce qui permet à
// l'interface de l'appeler à chaque affichage de bouton sans coût.
func (p *Provider) Available() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.echec != nil {
		return false
	}
	if p.session != nil {
		return true
	}
	// Le runtime est réévalué : l'utilisateur peut déposer la DLL pendant que
	// l'application tourne, comme il peut déposer le modèle.
	if p.runtime == "" {
		p.runtime = TrouverRuntime(p.dir)
	}
	return p.runtime != "" && model.Detect(p.dir).Available
}

// Embed vectorise un texte destiné à l'index.
func (p *Provider) Embed(texte string) ([]float32, error) {
	return p.vectoriser(prefixePassage + texte)
}

// EmbedQuery vectorise une requête.
func (p *Provider) EmbedQuery(texte string) ([]float32, error) {
	return p.vectoriser(prefixeRequete + texte)
}

func (p *Provider) vectoriser(texte string) ([]float32, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.charger(); err != nil {
		return nil, err
	}

	ids := p.tok.Encode(texte)
	n := int64(len(ids))

	// Une seule séquence, sans remplissage : le masque d'attention vaut 1
	// partout et les types de segment 0 partout. On les fournit quand même,
	// parce que le graphe les déclare en entrée.
	masque := make([]int64, n)
	for i := range masque {
		masque[i] = 1
	}
	types := make([]int64, n)

	forme := ort.NewShape(1, n)
	tIDs, err := ort.NewTensor(forme, ids)
	if err != nil {
		return nil, fmt.Errorf("tenseur des identifiants : %w", err)
	}
	defer tIDs.Destroy()
	tMasque, err := ort.NewTensor(forme, masque)
	if err != nil {
		return nil, fmt.Errorf("tenseur du masque : %w", err)
	}
	defer tMasque.Destroy()
	tTypes, err := ort.NewTensor(forme, types)
	if err != nil {
		return nil, fmt.Errorf("tenseur des types : %w", err)
	}
	defer tTypes.Destroy()

	// La sortie est laissée à nil : la session l'alloue à la bonne forme, ce
	// qui évite d'inscrire la dimension cachée en dur ici.
	sorties := []ort.Value{nil}
	if err := p.session.Run([]ort.Value{tIDs, tMasque, tTypes}, sorties); err != nil {
		return nil, fmt.Errorf("inférence : %w", err)
	}
	defer sorties[0].Destroy()

	etats, ok := sorties[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("sortie de type inattendu : %T", sorties[0])
	}
	return moyennePuisNormalise(etats.GetData(), etats.GetShape())
}

// moyennePuisNormalise transforme la sortie du modèle en un vecteur unique.
//
// C'est la recette de la famille e5 : moyenne des états cachés sur les jetons,
// puis normalisation L2. Prendre le seul jeton <s>, comme le ferait un modèle à
// « pooler », donnerait des vecteurs nettement moins bons — e5 n'est pas
// entraîné ainsi.
//
// La normalisation n'est pas une élégance : elle rend le produit scalaire égal
// à la similarité cosinus, ce qui garde les scores comparables d'un texte à
// l'autre quelle que soit sa longueur.
func moyennePuisNormalise(data []float32, forme ort.Shape) ([]float32, error) {
	if len(forme) != 3 {
		return nil, fmt.Errorf("sortie de forme %v, attendu [lot, jetons, dimensions]", forme)
	}
	jetons, dims := int(forme[1]), int(forme[2])
	if jetons == 0 || dims == 0 || len(data) < jetons*dims {
		return nil, fmt.Errorf("sortie incohérente : forme %v pour %d valeurs", forme, len(data))
	}

	vecteur := make([]float64, dims)
	for j := 0; j < jetons; j++ {
		ligne := data[j*dims : (j+1)*dims]
		for d := 0; d < dims; d++ {
			vecteur[d] += float64(ligne[d])
		}
	}

	var norme float64
	for d := range vecteur {
		vecteur[d] /= float64(jetons)
		norme += vecteur[d] * vecteur[d]
	}
	if norme == 0 {
		return nil, fmt.Errorf("vecteur de norme nulle")
	}
	norme = math.Sqrt(norme)

	out := make([]float32, dims)
	for d := range vecteur {
		out[d] = float32(vecteur[d] / norme)
	}
	return out, nil
}

// charger initialise le runtime, le tokenizer et la session, une seule fois.
//
// Appelée sous verrou par vectoriser.
func (p *Provider) charger() error {
	if p.echec != nil {
		return p.echec
	}
	if p.session != nil {
		return nil
	}

	statut := model.Detect(p.dir)
	if !statut.Available {
		// Modèle non déposé : ce n'est pas une panne, c'est l'état de départ.
		// On ne mémorise donc **pas** l'échec — l'utilisateur peut déposer les
		// fichiers sans redémarrer.
		return domain.ErrModelUnavailable
	}
	if p.runtime == "" {
		p.runtime = TrouverRuntime(p.dir)
	}
	if p.runtime == "" {
		return fmt.Errorf("%w : %s est introuvable", domain.ErrModelUnavailable, NomRuntime)
	}

	log.Printf("[semantique] chargement du modele depuis %s", p.dir)

	if !ort.IsInitialized() {
		ort.SetSharedLibraryPath(p.runtime)
		if err := ort.InitializeEnvironment(); err != nil {
			// Une DLL absente ou incompatible ne se répare pas en réessayant :
			// on retient l'échec pour ne pas le retenter à chaque frappe.
			p.echec = fmt.Errorf("initialisation d'ONNX Runtime (%s) : %w", p.runtime, err)
			return p.echec
		}
	}

	cheminModele, cheminTokenizer, _ := model.Paths(p.dir)

	tok, err := chargerTokenizer(cheminTokenizer)
	if err != nil {
		p.echec = err
		return p.echec
	}

	session, err := ort.NewDynamicAdvancedSession(cheminModele, entrees, sortie, nil)
	if err != nil {
		p.echec = fmt.Errorf("ouverture du modèle : %w", err)
		return p.echec
	}

	p.tok, p.session = tok, session
	log.Printf("[semantique] modele charge : %d pieces de vocabulaire", len(tok.pieces))
	return nil
}

// Close libère la session et le runtime.
//
// Appelée à l'arrêt de l'application. Sans effet si rien n'a jamais été chargé,
// ce qui est le cas courant.
func (p *Provider) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.session != nil {
		p.session.Destroy()
		p.session = nil
	}
	p.tok = nil
	if ort.IsInitialized() {
		if err := ort.DestroyEnvironment(); err != nil {
			log.Printf("[semantique] fermeture d'ONNX Runtime : %v", err)
		}
	}
}
