import { sheetStyle } from '../lib/production';
import { cn } from '../lib/cn';

// SheetBadge — метка отработки: номер листа в цвете, детерминированном по
// id. Все заказы одной партии несут одинаковый бейдж — группа считывается
// цветом. Рендерит ничего без sheetId (заказ без отработки).
export function SheetBadge({ sheetId, deviations = 0, showStatus = false, className = '' }) {
  if (!sheetId) return null;
  const style = sheetStyle(sheetId);
  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center gap-1 rounded-full border px-1.5 py-0.5 text-[11px] font-semibold leading-4 tabular-nums',
        style.badge,
        className,
      )}
      title={deviations > 0 ? `Отработан с отклонениями · лист №${sheetId} · отклонений: ${deviations}` : `Отработан по заявке · лист №${sheetId}`}
      aria-label={deviations > 0 ? `Отработан с отклонениями, лист номер ${sheetId}, отклонений ${deviations}` : `Отработан по заявке, лист номер ${sheetId}`}
    >
      <span className={cn('h-1.5 w-1.5 rounded-full', style.dot)} aria-hidden="true" />
      {showStatus ? (
        <>
          <span aria-hidden="true">{deviations > 0 ? '±' : '✓'}</span>
          <span>{deviations > 0 ? `Отклонения ${deviations}` : 'Отработан'}</span>
          <span className="opacity-70">№{sheetId}</span>
        </>
      ) : (
        <>
          <span aria-hidden="true">{deviations > 0 ? `±${deviations}` : '✓'}</span>
          №{sheetId}
        </>
      )}
    </span>
  );
}
