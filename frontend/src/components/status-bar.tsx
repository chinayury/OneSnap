import { CaptureResult } from '../types';
import { useT } from '../i18n';

interface StatusBarProps {
  screenshot: CaptureResult | null;
  message?: string;
}

export function StatusBar({ screenshot, message }: StatusBarProps) {
  const t = useT();
  return (
    <div className="px-4 py-2 glass text-sm flex justify-between items-center">
      <span className="text-slate-400">
        {message ? (
          <span className="text-cyan-400">{message}</span>
        ) : screenshot ? (
          <span className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
            <span className="text-slate-300">
              {t('status.image')}: <span className="text-violet-400 font-medium">{screenshot.width}</span>
              <span className="text-slate-500"> × </span>
              <span className="text-violet-400 font-medium">{screenshot.height}</span>
            </span>
          </span>
        ) : (
          <span className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-slate-500" />
            {t('status.ready')}
          </span>
        )}
      </span>
      <span className="flex items-center gap-3">
        <span className="text-gradient font-semibold">OneSnap v{__APP_VERSION__}</span>
        <span className="text-slate-500">•</span>
        <a
          href="https://github.com/chinayury/OneSnap"
          target="_blank"
          rel="noopener noreferrer"
          className="text-slate-400 hover:text-cyan-400 transition-colors"
        >
          GitHub
        </a>
      </span>
    </div>
  );
}
