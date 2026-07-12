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
        'ui-number inline-flex min-w-0 max-w-full items-center gap-1 overflow-hidden text-caption font-semibold leading-4',
        className,
      )}
      title={deviations > 0 ? `Отработан с отклонениями · лист №${sheetId} · отклонений: ${deviations}` : `Отработан по заявке · лист №${sheetId}`}
      aria-label={deviations > 0 ? `Отработан с отклонениями, лист номер ${sheetId}, отклонений ${deviations}` : `Отработан по заявке, лист номер ${sheetId}`}
    >
      <span className={cn('h-1.5 w-1.5 shrink-0 rounded-full', style.dot)} aria-hidden="true" />
      {showStatus ? (
        <>
          {/* На узких карточках матрицы (3 колонки на телефоне) слово не
              помещается — остаются символ и номер, полный текст в title. */}
          <span aria-hidden="true">{deviations > 0 ? `±${deviations}` : '✓'}</span>
          <span className="hidden truncate sm:inline">{deviations > 0 ? 'Отклонения' : 'Отработан'}</span>
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
