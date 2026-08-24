import Image from '@tiptap/extension-image'
import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { useEffect } from 'react'

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
}: {
  content: string
  onChange: (html: string) => void
  onBlur?: () => void
  readOnly?: boolean
}) {
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

  // Quand on change de note, TipTap garde le document précédent : il faut le
  // remplacer explicitement. `emitUpdate: false` évite que ce remplacement soit
  // pris pour une frappe de l'utilisateur et déclenche une sauvegarde — ce qui
  // écraserait la note qu'on vient d'ouvrir avec le contenu de la précédente.
  useEffect(() => {
    if (editor && content !== editor.getHTML()) {
      editor.commands.setContent(content, { emitUpdate: false })
    }
  }, [editor, content])

  return (
    <div className="flex h-full flex-col">
      {!readOnly && editor ? (
        <div role="toolbar" aria-label="Mise en forme" className="flex flex-wrap gap-1 border-b border-neutral-300 px-2 py-1">
          <Outil editor={editor} actif="bold" label="Gras" onClick={() => editor.chain().focus().toggleBold().run()}>
            <strong>G</strong>
          </Outil>
          <Outil editor={editor} actif="italic" label="Italique" onClick={() => editor.chain().focus().toggleItalic().run()}>
            <em>I</em>
          </Outil>
          <Outil editor={editor} actif="strike" label="Barré" onClick={() => editor.chain().focus().toggleStrike().run()}>
            <s>S</s>
          </Outil>
          <span aria-hidden className="mx-1 h-5 w-px self-center bg-neutral-300" />
          <Outil editor={editor} actif="heading" attrs={{ level: 2 }} label="Titre" onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}>
            T
          </Outil>
          <Outil editor={editor} actif="bulletList" label="Liste à puces" onClick={() => editor.chain().focus().toggleBulletList().run()}>
            •
          </Outil>
          <Outil editor={editor} actif="orderedList" label="Liste numérotée" onClick={() => editor.chain().focus().toggleOrderedList().run()}>
            1.
          </Outil>
          <Outil editor={editor} actif="codeBlock" label="Bloc de code" onClick={() => editor.chain().focus().toggleCodeBlock().run()}>
            {'</>'}
          </Outil>
          <span aria-hidden className="mx-1 h-5 w-px self-center bg-neutral-300" />
          <Outil editor={editor} actif="link" label="Lien" onClick={() => poserLien(editor)}>
            🔗
          </Outil>
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

/** Bouton de la barre d'outils, dont l'état actif suit la sélection courante. */
function Outil({
  editor,
  actif,
  attrs,
  label,
  onClick,
  children,
}: {
  editor: NonNullable<ReturnType<typeof useEditor>>
  actif: string
  attrs?: Record<string, unknown>
  label: string
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      aria-pressed={editor.isActive(actif, attrs)}
      // `onMouseDown` plutôt que `onClick` : sans cela le bouton vole le focus
      // à l'éditeur, la sélection se perd et la commande s'applique dans le vide.
      onMouseDown={(e) => {
        e.preventDefault()
        onClick()
      }}
      className={`min-w-7 rounded px-1.5 py-0.5 text-xs ${
        editor.isActive(actif, attrs) ? 'border-2 border-neutral-900' : 'border border-neutral-300'
      }`}
    >
      {children}
    </button>
  )
}
