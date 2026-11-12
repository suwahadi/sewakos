export function LegalPage({ title, updated, paragraphs }: { title: string; updated: string; paragraphs: string[] }) {
  return (
    <article className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h1 className="text-title-lg font-semibold text-neutral-900">{title}</h1>
        <p className="text-caption text-neutral-600">Terakhir diperbarui {updated}</p>
      </div>
      {paragraphs.map((paragraph) => (
        <p key={paragraph.slice(0, 32)} className="text-body leading-relaxed text-neutral-800">
          {paragraph}
        </p>
      ))}
    </article>
  );
}
