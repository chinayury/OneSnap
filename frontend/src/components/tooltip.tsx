import { useCallback, useEffect, useRef, useState, ReactNode } from 'react';
import { createPortal } from 'react-dom';

interface TooltipProps {
  text: string;
  children: ReactNode;
  /** Prefer showing above the target (used for toolbars near screen bottom) */
  side?: 'top' | 'bottom';
}

/**
 * Lightweight styled tooltip shown on hover (350ms delay).
 * Renders through a portal so it is not clipped by overflow containers.
 * Replaces inconsistent native `title` attributes across the app.
 */
export function Tooltip({ text, children, side = 'top' }: TooltipProps) {
  const [visible, setVisible] = useState(false);
  const [pos, setPos] = useState({ top: 0, left: 0 });
  const anchorRef = useRef<HTMLSpanElement>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearTimer = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const handleEnter = useCallback(() => {
    clearTimer();
    timerRef.current = setTimeout(() => {
      const el = anchorRef.current;
      if (!el) return;
      const rect = el.getBoundingClientRect();
      // Horizontal center on the target, clamped to viewport
      let left = rect.left + rect.width / 2;
      const maxTextWidth = 320;
      left = Math.max(maxTextWidth / 2 + 8, Math.min(window.innerWidth - maxTextWidth / 2 - 8, left));
      // Vertical: requested side, flip if it would leave the viewport
      const approxHeight = 36;
      let top = side === 'top' ? rect.top - approxHeight - 6 : rect.bottom + 6;
      if (side === 'top' && top < 4) top = rect.bottom + 6;
      if (side === 'bottom' && top + approxHeight > window.innerHeight - 4) top = rect.top - approxHeight - 6;
      setPos({ top, left });
      setVisible(true);
    }, 350);
  }, [clearTimer, side]);

  const handleLeave = useCallback(() => {
    clearTimer();
    setVisible(false);
  }, [clearTimer]);

  useEffect(() => () => clearTimer(), [clearTimer]);

  return (
    <>
      <span
        ref={anchorRef}
        onMouseEnter={handleEnter}
        onMouseLeave={handleLeave}
        onMouseDown={handleLeave}
        style={{ display: 'inline-flex' }}
      >
        {children}
      </span>
      {visible && createPortal(
        <div
          style={{
            position: 'fixed',
            top: pos.top,
            left: pos.left,
            transform: 'translateX(-50%)',
            zIndex: 100000,
            maxWidth: 320,
            padding: '6px 10px',
            borderRadius: 8,
            background: 'rgba(15, 23, 42, 0.96)',
            border: '1px solid rgba(139, 92, 246, 0.35)',
            color: '#e2e8f0',
            fontSize: 12,
            lineHeight: 1.5,
            whiteSpace: 'normal',
            textAlign: 'center',
            pointerEvents: 'none',
            boxShadow: '0 4px 16px rgba(0, 0, 0, 0.4)',
            backdropFilter: 'blur(6px)',
          }}
        >
          {text}
        </div>,
        document.body
      )}
    </>
  );
}
