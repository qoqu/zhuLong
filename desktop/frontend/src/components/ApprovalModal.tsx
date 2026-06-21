import type { ApprovalRequest, Language } from '../types'
import { useT } from '../i18n'

interface ApprovalModalProps {
  language: Language
  request: ApprovalRequest
  onApprove: () => void
  onDeny: () => void
  onAlwaysAllow: () => void
}

const riskLabel = (l: Language, r: string) => {
  if (l === 'en') return r === 'high' ? 'High Risk' : r === 'medium' ? 'Medium Risk' : 'Low Risk'
  return r === 'high' ? '高风险' : r === 'medium' ? '中风险' : '低风险'
}

export function ApprovalModal(props: ApprovalModalProps) {
  const t = useT(props.language)
  const r = props.request
  const args = Object.entries(r.args || {})

  return (
    <div className="modal-backdrop">
      <div className="modal" role="dialog" aria-modal="true" aria-labelledby="approval-title">
        <div className="modal__header">
          <div className="modal__icon">🛡</div>
          <div className="modal__title-block">
            <h2 id="approval-title" className="modal__title">
              {t.approvalTitle}
            </h2>
            <div className={`modal__risk modal__risk--${r.risk}`}>
              <span className="modal__risk-dot" />
              <span>{riskLabel(props.language, r.risk)}</span>
            </div>
          </div>
          <button
            className="modal__close"
            onClick={props.onDeny}
            title="Close"
            aria-label="Close"
          >
            ✕
          </button>
        </div>

        <div className="modal__body">
          <div className="modal__row">
            <span className="modal__label">{t.approvalTool}</span>
            <code className="modal__tool">{r.tool}</code>
          </div>

          {r.reason && (
            <div className="modal__row">
              <span className="modal__label">{t.approvalReason}</span>
              <p className="modal__reason">{r.reason}</p>
            </div>
          )}

          {args.length > 0 && (
            <div className="modal__row">
              <span className="modal__label">{t.approvalArgs}</span>
              <div className="modal__args">
                {args.map(([k, v]) => (
                  <div key={k} className="modal__arg">
                    <span className="modal__arg-key">{k}</span>
                    <code className="modal__arg-val">{formatValue(v)}</code>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        <div className="modal__footer">
          <button className="modal__btn modal__btn--ghost" onClick={props.onDeny}>
            {t.deny}
          </button>
          <button className="modal__btn modal__btn--secondary" onClick={props.onAlwaysAllow}>
            {t.alwaysAllow}
          </button>
          <button className="modal__btn modal__btn--primary" onClick={props.onApprove} autoFocus>
            {t.approve}
          </button>
        </div>
      </div>
    </div>
  )
}

function formatValue(v: any): string {
  if (v === null || v === undefined) return ''
  if (typeof v === 'string') return v
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return String(v)
  }
}
