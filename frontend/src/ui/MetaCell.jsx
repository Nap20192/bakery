export function MetaCell({ label, value }) {
  return (
    <div className="ui-meta-cell">
      <span className="ui-meta-cell__label">{label}</span>
      <strong className="ui-meta-cell__value">{value}</strong>
    </div>
  );
}
