import Image from '@tiptap/extension-image'
import { EditorContent, useEditor, useEditorState } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { useEffect, useRef, useState } from 'react'
import { COULEURS, teinte } from '../couleurs'

type ViewMode = 'inline' | 'readonly-full' | 'editable-full'

/** Taille maximale d'une image collée, avant refus (§3.3). */
const TAILLE_IMAGE_MAX = 10 * 1024 * 1024

/**
 * Éditeur riche (§3.4), adossé à TipTap.
 *
 * TipTap produit du HTML, stocké tel quel dans les colonnes TEXT de SQLite :
 * c'est le seul point de contact avec le backend, et il ne dépend ni de Go ni
 * de Wails.
 */
export function RichEditor({
  content,
  onChange,
  onBlur,
  readOnly,
  color,
  onColorChange,
}: {
  content: string
  onChange: (html: string) => void
  onBlur?: () => void
  readOnly?: boolean
  /**
   * Couleur de ce que l'éditeur édite — une note, ou la réunion dont on édite
   * le compte rendu (v1.2.0). L'éditeur ne sait pas de quoi il s'agit : il
   * montre le bouton de couleur si `onColorChange` est fourni, et c'est tout.
   */
  color?: string
  onColorChange?: (color: string) => void
}) {
  const [viewMode, setViewMode] = useState<ViewMode>('inline')
  const [paletteOuverte, setPaletteOuverte] = useState(false)
  const paletteRef = useRef<HTMLSpanElement>(null)
  // Lu par le gestionnaire d'Échap du plein écran, posé avant celui de la
  // palette : Échap doit fermer la palette, pas le plein écran derrière elle.
  const paletteOuverteRef = useRef(false)
  paletteOuverteRef.current = paletteOuverte

  const editor = useEditor({
    extensions: [
      StarterKit,
      // allowBase64 : une image collée est intégrée au HTML de la note. C'est
      // ce qui garantit qu'elle survit hors ligne, sans dépendre d'un serveur
      // (§3.3). Le basculement vers un fichier au-delà de 50 Ko viendra avec le
      // stockage d'images ; pour l'instant tout est en ligne.
      Image.configure({ allowBase64: true }),
    ],
    content,
    editable: !readOnly,
    onUpdate: ({ editor }) => onChange(editor.getHTML()),
    onBlur: () => onBlur?.(),
    editorProps: {
      attributes: { class: 'min-h-full p-3' },

      /**
       * Collage d'une image présente dans le presse-papier (§3.4).
       *
       * On ne traite que les images **binaires** — une capture d'écran, ou une
       * image copiée depuis un navigateur. Elles sont converties en data URI et
       * insérées, donc disponibles hors ligne.
       */
      handlePaste(view, event) {
        const fichiers = Array.from(event.clipboardData?.files ?? [])
        const image = fichiers.find((f) => f.type.startsWith('image/'))
        if (!image) return false

        if (image.size > TAILLE_IMAGE_MAX) {
          // Refus explicite plutôt que collage silencieux d'un fichier énorme,
          // qui gonflerait la base sans que personne ne comprenne pourquoi.
          window.alert("L'image dépasse 10 Mo et n'a pas été collée.")
          return true
        }

        const lecteur = new FileReader()
        lecteur.onload = () => {
          const src = String(lecteur.result)
          const node = view.state.schema.nodes.image?.create({ src })
          if (!node) return
          view.dispatch(view.state.tr.replaceSelectionWith(node))
        }
        lecteur.readAsDataURL(image)
        return true // on a pris la main : ProseMirror ne doit pas coller en plus
      },

      /**
       * Nettoyage du HTML collé (§3.4).
       *
       * Les images référencées par une URL distante sont **retirées**, pas
       * rapatriées : l'application n'accède jamais au réseau (§6). Les laisser
       * produirait des images cassées dès la première ouverture hors ligne,
       * ce qui est pire qu'une absence visible.
       */
      transformPastedHTML(html) {
        return html.replace(/<img\b[^>]*>/gi, (balise) =>
          /src\s*=\s*["']data:/i.test(balise) ? balise : '',
        )
      },
    },
  })

  // État « enfoncé » des boutons, recalculé à chaque transaction — y compris un
  // simple déplacement du curseur. TipTap 3 ne relance pas le rendu sur une
  // transaction ; sans cet abonnement, la barre ne suivait qu'à la frappe
  // (correctif 1.1.1). Le rendu ne repart que si l'un des booléens change.
  const actifs = useEditorState({
    editor,
    selector: ({ editor: e }) => ({
      bold: !!e?.isActive('bold'),
      italic: !!e?.isActive('italic'),
      strike: !!e?.isActive('strike'),
      heading: !!e?.isActive('heading', { level: 2 }),
      bulletList: !!e?.isActive('bulletList'),
      orderedList: !!e?.isActive('orderedList'),
      codeBlock: !!e?.isActive('codeBlock'),
      link: !!e?.isActive('link'),
    }),
  })

  // Quand on change de note, TipTap garde le document précédent : il faut le
  // remplacer explicitement. `emitUpdate: false` évite que ce remplacement soit
  // pris pour une frappe de l'utilisateur et déclenche une sauvegarde — ce qui
  // écraserait la note qu'on vient d'ouvrir avec le contenu de la précédente.
  useEffect(() => {
    if (editor && content !== editor.getHTML()) {
      editor.commands.setContent(content, { emitUpdate: false })
    }
  }, [editor, content])

  // Ferme une vue plein écran : l'éditabilité est recalculée depuis le prop
  // `readOnly` courant, pas depuis une valeur mise en cache lors de
  // l'ouverture du mode lecture seule (§2.2).
  function fermerPleinEcran() {
    editor?.setEditable(!readOnly)
    setViewMode('inline')
  }

  // Actif seulement pendant qu'un mode plein écran est ouvert, pour ne pas
  // interférer avec d'autres gestionnaires d'Échap (ex. ContextMenu) en mode inline.
  useEffect(() => {
    if (viewMode === 'inline') return
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !paletteOuverteRef.current) fermerPleinEcran()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [viewMode])

  // La palette se referme sans rien changer sur Échap ou sur un clic ailleurs.
  useEffect(() => {
    if (!paletteOuverte) return
    const onPointer = (event: MouseEvent) => {
      if (!paletteRef.current?.contains(event.target as Node)) setPaletteOuverte(false)
    }
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setPaletteOuverte(false)
    }
    window.addEventListener('mousedown', onPointer, true)
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('mousedown', onPointer, true)
      window.removeEventListener('keydown', onKey)
    }
  }, [paletteOuverte])

  function choisirCouleur(cle: string) {
    setPaletteOuverte(false)
    if (cle !== (color ?? '')) onColorChange?.(cle)
  }

  const barreOutilsVisible = !readOnly && !!editor && viewMode !== 'readonly-full'
  const pleinEcran = viewMode !== 'inline'

  return (
    <div className={pleinEcran ? 'fixed inset-0 z-50 flex flex-col bg-white' : 'flex h-full flex-col'}>
      {barreOutilsVisible ? (
        <div role="toolbar" aria-label="Mise en forme" className="flex flex-wrap gap-1 border-b border-neutral-300 px-2 py-1">
          <Outil pressed={!!actifs?.bold} label="Gras" onClick={() => editor.chain().focus().toggleBold().run()}>
            <strong>G</strong>
          </Outil>
          <Outil pressed={!!actifs?.italic} label="Italique" onClick={() => editor.chain().focus().toggleItalic().run()}>
            <em>I</em>
          </Outil>
          <Outil pressed={!!actifs?.strike} label="Barré" onClick={() => editor.chain().focus().toggleStrike().run()}>
            <s>S</s>
          </Outil>
          <span aria-hidden className="mx-1 h-5 w-px self-center bg-neutral-300" />
          <Outil pressed={!!actifs?.heading} label="Titre" onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}>
            T
          </Outil>
          <Outil pressed={!!actifs?.bulletList} label="Liste à puces" onClick={() => editor.chain().focus().toggleBulletList().run()}>
            •
          </Outil>
          <Outil pressed={!!actifs?.orderedList} label="Liste numérotée" onClick={() => editor.chain().focus().toggleOrderedList().run()}>
            1.
          </Outil>
          <Outil pressed={!!actifs?.codeBlock} label="Bloc de code" onClick={() => editor.chain().focus().toggleCodeBlock().run()}>
            {'</>'}
          </Outil>
          <span aria-hidden className="mx-1 h-5 w-px self-center bg-neutral-300" />
          <Outil pressed={!!actifs?.link} label="Lien" onClick={() => poserLien(editor)}>
            🔗
          </Outil>
          <span aria-hidden className="mx-1 h-5 w-px self-center bg-neutral-300" />
          <button
            type="button"
            title="Afficher en plein écran (lecture seule)"
            aria-label="Afficher en plein écran (lecture seule)"
            onMouseDown={(e) => {
              e.preventDefault()
              editor.setEditable(false)
              setViewMode('readonly-full')
            }}
            className="min-w-7 rounded border border-neutral-300 px-1.5 py-0.5 text-xs"
          >
            👁
          </button>
          <button
            type="button"
            title="Agrandir en plein écran"
            aria-label="Agrandir en plein écran"
            onMouseDown={(e) => {
              e.preventDefault()
              setViewMode('editable-full')
            }}
            className="min-w-7 rounded border border-neutral-300 px-1.5 py-0.5 text-xs"
          >
            ⛶
          </button>
          {onColorChange ? (
            <span ref={paletteRef} className="relative">
              <button
                type="button"
                title="Couleur"
                aria-label="Couleur"
                aria-haspopup="menu"
                aria-expanded={paletteOuverte}
                onMouseDown={(e) => {
                  e.preventDefault()
                  setPaletteOuverte((ouverte) => !ouverte)
                }}
                style={{ backgroundColor: teinte(color) }}
                className="min-w-7 rounded border border-neutral-300 px-1.5 py-0.5 text-xs"
              >
                🎨
              </button>
              {paletteOuverte ? (
                <div
                  role="menu"
                  aria-label="Couleurs"
                  className="absolute left-0 top-full z-50 mt-1 grid w-max grid-cols-6 gap-1 rounded border border-neutral-300 bg-white p-1.5 shadow-lg"
                >
                  {COULEURS.map((c) => (
                    <Pastille key={c.cle} nom={c.nom} hex={c.hex} choisie={color === c.cle} onChoisir={() => choisirCouleur(c.cle)} />
                  ))}
                  <Pastille nom="Aucune couleur" choisie={!teinte(color)} onChoisir={() => choisirCouleur('')} />
                </div>
              ) : null}
            </span>
          ) : null}
          {pleinEcran ? (
            <button
              type="button"
              title="Fermer le plein écran"
              aria-label="Fermer le plein écran"
              onMouseDown={(e) => {
                e.preventDefault()
                fermerPleinEcran()
              }}
              className="ml-auto min-w-7 rounded border border-neutral-300 px-1.5 py-0.5 text-xs"
            >
              ✕
            </button>
          ) : null}
        </div>
      ) : null}
      {!barreOutilsVisible && viewMode === 'readonly-full' ? (
        <div className="flex justify-end border-b border-neutral-300 px-2 py-1">
          <button
            type="button"
            title="Fermer le plein écran"
            aria-label="Fermer le plein écran"
            onClick={fermerPleinEcran}
            className="min-w-7 rounded border border-neutral-300 px-1.5 py-0.5 text-xs"
          >
            ✕
          </button>
        </div>
      ) : null}
      <div className="min-h-0 flex-1 overflow-auto">
        <EditorContent editor={editor} className="h-full" />
      </div>
    </div>
  )
}

/** Pose ou retire un lien sur la sélection courante. */
function poserLien(editor: NonNullable<ReturnType<typeof useEditor>>) {
  if (editor.isActive('link')) {
    editor.chain().focus().unsetLink().run()
    return
  }
  const url = window.prompt('Adresse du lien ?')
  if (!url?.trim()) return
  editor.chain().focus().setLink({ href: url.trim() }).run()
}

/** Pastille de la palette de couleurs. Sans `hex`, c'est le choix « aucune couleur ». */
function Pastille({
  nom,
  hex,
  choisie,
  onChoisir,
}: {
  nom: string
  hex?: string
  choisie: boolean
  onChoisir: () => void
}) {
  return (
    <button
      type="button"
      role="menuitemradio"
      aria-checked={choisie}
      aria-label={nom}
      title={nom}
      // `onMouseDown`, comme les autres boutons de la barre : l'éditeur garde le focus.
      onMouseDown={(e) => {
        e.preventDefault()
        onChoisir()
      }}
      style={{ backgroundColor: hex }}
      className={`flex h-5 w-5 items-center justify-center rounded-full text-[10px] leading-none text-neutral-500 ${
        choisie ? 'border-2 border-neutral-900' : 'border border-neutral-400'
      } ${hex ? '' : 'bg-white'}`}
    >
      {hex ? null : '∅'}
    </button>
  )
}

/** Bouton de la barre d'outils ; `pressed` vient de l'état de l'éditeur à la sélection courante. */
function Outil({
  pressed,
  label,
  onClick,
  children,
}: {
  pressed: boolean
  label: string
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      aria-pressed={pressed}
      // `onMouseDown` plutôt que `onClick` : sans cela le bouton vole le focus
      // à l'éditeur, la sélection se perd et la commande s'applique dans le vide.
      onMouseDown={(e) => {
        e.preventDefault()
        onClick()
      }}
      className={`min-w-7 rounded px-1.5 py-0.5 text-xs ${
        pressed ? 'border-2 border-neutral-900' : 'border border-neutral-300'
      }`}
    >
      {children}
    </button>
  )
}
