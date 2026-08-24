import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { useEffect } from 'react'

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
    extensions: [StarterKit],
    content,
    editable: !readOnly,
    onUpdate: ({ editor }) => onChange(editor.getHTML()),
    onBlur: () => onBlur?.(),
    editorProps: {
      attributes: {
        class: 'prose prose-sm max-w-none min-h-full p-3 focus:outline-none',
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
        </div>
      ) : null}
      <div className="min-h-0 flex-1 overflow-auto">
        <EditorContent editor={editor} className="h-full" />
      </div>
    </div>
  )
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
