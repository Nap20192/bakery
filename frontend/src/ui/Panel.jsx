// Panel — shadcn/ui Card на семантических токенах. panelClass применяется к
// контейнеру напрямую (исторический API); PanelHeader — шапка с надзаголовком
// и счётчиком позиций.
export const panelClass = 'ui-panel';

export function PanelHeader({ eyebrow, title, count }) {
  return (
    <div className="ui-panel-header">
      <div className="ui-panel-header__copy">
        {eyebrow && <span className="ui-panel-header__eyebrow">{eyebrow}</span>}
        <h3 className="ui-panel-header__title">{title}</h3>
      </div>
      {count !== undefined && (
        <span className="ui-panel-header__count">
          {count} поз.
        </span>
      )}
    </div>
  );
}
