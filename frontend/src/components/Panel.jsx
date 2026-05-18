export const panelClass = 'rounded-lg border border-stone-300 bg-[#fff7df] p-3 sm:p-4';

export function PanelHeader({ eyebrow, title, count }) {
  return (
    <div className="mb-2.5 flex items-start justify-between gap-3">
      <div className="min-w-0">
        {eyebrow && <span className="text-[11px] font-medium uppercase text-stone-600">{eyebrow}</span>}
        <h3 className="m-0 break-words text-[17px] font-semibold leading-6 text-stone-950 sm:text-[18px]">{title}</h3>
      </div>
      {count !== undefined && (
        <span className="shrink-0 rounded-full border border-stone-300 bg-[#fff7df] px-2 py-1 text-[11px] font-medium text-stone-600 sm:text-xs">
          {count} поз.
        </span>
      )}
    </div>
  );
}
