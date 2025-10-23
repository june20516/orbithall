import { useI18n } from "../i18n/context";

interface ErrorOverlayProps {
  error: string;
  onClose: () => void;
}

export function ErrorOverlay({ error, onClose }: ErrorOverlayProps) {
  const { t } = useI18n();

  return (
    <div className="orb-error-overlay">
      <div className="orb-error-backdrop" onClick={onClose} />
      <div className="orb-error-container">
        <div className="orb-error-content">{error}</div>
        <button
          className="orb-error-close"
          onClick={onClose}
          aria-label={t("error.close")}
        >
          ×
        </button>
      </div>
    </div>
  );
}
