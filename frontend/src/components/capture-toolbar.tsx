import { CaptureMode } from '../types';
import { Monitor, Scan, AppWindow, Settings, ChevronDown, FolderOpen, Clipboard } from 'lucide-react';
import { Tooltip } from './tooltip';
import { useT } from '../i18n';

interface CaptureToolbarProps {
  onCapture: (mode: CaptureMode) => void;
  isCapturing: boolean;
  hasScreenshot: boolean;
  onClear: () => void;
  onMinimize?: () => void;
  onOpenSettings?: () => void;
  onImportImage?: () => void;
  onClipboardCapture?: () => void;
}

export function CaptureToolbar({ onCapture, isCapturing, hasScreenshot, onClear, onMinimize, onOpenSettings, onImportImage, onClipboardCapture }: CaptureToolbarProps) {
  const t = useT();
  return (
    <div className="flex items-center gap-4 px-4 py-3 glass">
      <div className="flex gap-2">
        {/* Fullscreen - Primary gradient button */}
        <Tooltip text={t('tip.fullscreen')} side="bottom">
          <button
            onClick={() => onCapture('fullscreen')}
            disabled={isCapturing}
            className="px-4 py-2.5 rounded-xl transition-all duration-200 flex items-center gap-2
                       bg-gradient-to-r from-violet-500 to-purple-600 hover:from-violet-400 hover:to-purple-500
                       disabled:from-slate-600 disabled:to-slate-700 disabled:opacity-50
                       text-white font-medium shadow-lg shadow-violet-500/25 hover:shadow-violet-500/40
                       hover:-translate-y-0.5 active:translate-y-0"
          >
            <Monitor className="w-4 h-4" />
            {t('capture.fullscreen')}
          </button>
        </Tooltip>

        {/* Region - Cyan/teal gradient */}
        <Tooltip text={t('tip.region')} side="bottom">
          <button
            onClick={() => onCapture('region')}
            disabled={isCapturing}
            className="px-4 py-2.5 rounded-xl transition-all duration-200 flex items-center gap-2
                       bg-gradient-to-r from-cyan-500 to-teal-500 hover:from-cyan-400 hover:to-teal-400
                       disabled:from-slate-600 disabled:to-slate-700 disabled:opacity-50
                       text-white font-medium shadow-lg shadow-cyan-500/25 hover:shadow-cyan-500/40
                       hover:-translate-y-0.5 active:translate-y-0"
          >
            <Scan className="w-4 h-4" />
            {t('capture.region')}
          </button>
        </Tooltip>

        {/* Window - Pink/rose gradient */}
        <Tooltip text={t('tip.window')} side="bottom">
          <button
            onClick={() => onCapture('window')}
            disabled={isCapturing}
            className="px-4 py-2.5 rounded-xl transition-all duration-200 flex items-center gap-2
                       bg-gradient-to-r from-pink-500 to-rose-500 hover:from-pink-400 hover:to-rose-400
                       disabled:from-slate-600 disabled:to-slate-700 disabled:opacity-50
                       text-white font-medium shadow-lg shadow-pink-500/25 hover:shadow-pink-500/40
                       hover:-translate-y-0.5 active:translate-y-0"
          >
            <AppWindow className="w-4 h-4" />
            {t('capture.window')}
          </button>
        </Tooltip>

        {/* Clipboard - Green/emerald gradient */}
        {onClipboardCapture && (
          <Tooltip text={t('tip.clipboard')} side="bottom">
            <button
              onClick={onClipboardCapture}
              disabled={isCapturing}
              className="px-4 py-2.5 rounded-xl transition-all duration-200 flex items-center gap-2
                         bg-gradient-to-r from-green-500 to-emerald-500 hover:from-green-400 hover:to-emerald-400
                         disabled:from-slate-600 disabled:to-slate-700 disabled:opacity-50
                         text-white font-medium shadow-lg shadow-green-500/25 hover:shadow-green-500/40
                         hover:-translate-y-0.5 active:translate-y-0"
            >
              <Clipboard className="w-4 h-4" />
              {t('capture.clipboard')}
            </button>
          </Tooltip>
        )}

        {/* Import - Amber/orange gradient */}
        {onImportImage && (
          <Tooltip text={t('tip.import')} side="bottom">
            <button
              onClick={onImportImage}
              disabled={isCapturing}
              className="px-4 py-2.5 rounded-xl transition-all duration-200 flex items-center gap-2
                         bg-gradient-to-r from-amber-500 to-orange-500 hover:from-amber-400 hover:to-orange-400
                         disabled:from-slate-600 disabled:to-slate-700 disabled:opacity-50
                         text-white font-medium shadow-lg shadow-amber-500/25 hover:shadow-amber-500/40
                         hover:-translate-y-0.5 active:translate-y-0"
            >
              <FolderOpen className="w-4 h-4" />
              {t('capture.import')}
            </button>
          </Tooltip>
        )}
      </div>

      {hasScreenshot && (
        <>
          <div className="h-6 w-px bg-white/10" />
          <Tooltip text={t('tip.clear')} side="bottom">
            <button
              onClick={onClear}
              className="px-4 py-2.5 rounded-xl transition-all duration-200
                         bg-white/5 hover:bg-white/10 border border-white/10 hover:border-white/20
                         text-slate-300 hover:text-white font-medium"
            >
              {t('capture.clear')}
            </button>
          </Tooltip>
        </>
      )}

      {isCapturing && (
        <div className="flex items-center gap-2">
          <div className="w-2 h-2 rounded-full bg-cyan-400 animate-pulse" />
          <span className="text-cyan-300 animate-pulse font-medium">{t('capture.capturing')}</span>
        </div>
      )}

      {/* Spacer */}
      <div className="flex-1" />

      {/* Settings button */}
      {onOpenSettings && (
        <Tooltip text={t('tip.settings')} side="bottom">
          <button
            onClick={onOpenSettings}
            className="p-2.5 rounded-xl text-slate-400 hover:text-white
                       bg-white/5 hover:bg-white/10 border border-white/5 hover:border-white/15
                       transition-all duration-200"
          >
            <Settings className="w-5 h-5" />
          </button>
        </Tooltip>
      )}

      {/* Minimize to tray button */}
      {onMinimize && (
        <Tooltip text={t('tip.minimize')} side="bottom">
          <button
            onClick={onMinimize}
            className="p-2.5 rounded-xl text-slate-400 hover:text-white
                       bg-white/5 hover:bg-white/10 border border-white/5 hover:border-white/15
                       transition-all duration-200"
          >
            <ChevronDown className="w-5 h-5" />
          </button>
        </Tooltip>
      )}
    </div>
  );
}
