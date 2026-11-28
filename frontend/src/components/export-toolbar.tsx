import type { ExportFormat } from '../types';
import { ClipboardCopy, Download, Save, Link, Image } from 'lucide-react';
import { Tooltip } from './tooltip';
import { useT } from '../i18n';

interface ExportToolbarProps {
  format: ExportFormat;
  onFormatChange: (format: ExportFormat) => void;
  onSave: () => void;
  onQuickSave: () => void;
  onCopyToClipboard: () => void;
  onCopyPath: () => void;
  onOpenLibrary: () => void;
  lastSavedPath: string | null;
  isExporting: boolean;
}

export function ExportToolbar({
  format,
  onFormatChange,
  onSave,
  onQuickSave,
  onCopyToClipboard,
  onCopyPath,
  onOpenLibrary,
  lastSavedPath,
  isExporting,
}: ExportToolbarProps) {
  const t = useT();

  return (
    <div className="flex items-center gap-3 px-3 py-2.5 glass">
      {/* Format Selection */}
      <div className="flex items-center gap-2 pr-3 border-r border-white/10">
        <span className="text-xs text-slate-400 font-medium">{t('export.format')}</span>
        <div className="flex gap-1">
          <button
            onClick={() => onFormatChange('png')}
            className={`px-3 py-1 text-xs rounded-lg font-medium transition-all duration-200 ${
              format === 'png'
                ? 'bg-gradient-to-r from-violet-500 to-purple-600 text-white shadow-lg shadow-violet-500/30'
                : 'bg-white/5 text-slate-300 hover:bg-white/10 border border-white/5'
            }`}
          >
            PNG
          </button>
          <button
            onClick={() => onFormatChange('jpeg')}
            className={`px-3 py-1 text-xs rounded-lg font-medium transition-all duration-200 ${
              format === 'jpeg'
                ? 'bg-gradient-to-r from-violet-500 to-purple-600 text-white shadow-lg shadow-violet-500/30'
                : 'bg-white/5 text-slate-300 hover:bg-white/10 border border-white/5'
            }`}
          >
            JPEG
          </button>
        </div>
      </div>

      {/* Export Actions */}
      <div className="flex items-center gap-2">
        {/* Copy to Clipboard */}
        <Tooltip text={t('tip.copy')}>
          <button
            onClick={onCopyToClipboard}
            disabled={isExporting}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg font-medium transition-all duration-200
                       bg-white/5 hover:bg-white/10 border border-white/10 hover:border-white/20
                       text-slate-300 hover:text-white
                       disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <ClipboardCopy className="w-4 h-4" />
            {t('export.copy')}
          </button>
        </Tooltip>

        {/* Copy Path - only show after save */}
        {lastSavedPath && (
          <Tooltip text={t('tip.copy.path')}>
            <button
              onClick={onCopyPath}
              disabled={isExporting}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg font-medium transition-all duration-200
                         bg-white/5 hover:bg-white/10 border border-white/10 hover:border-white/20
                         text-slate-300 hover:text-white
                         disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <Link className="w-4 h-4" />
              {t('export.copy.path')}
            </button>
          </Tooltip>
        )}

        {/* Quick Save */}
        <Tooltip text={t('tip.quick.save')}>
          <button
            onClick={onQuickSave}
            disabled={isExporting}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg font-medium transition-all duration-200
                       bg-gradient-to-r from-cyan-500/20 to-teal-500/20 hover:from-cyan-500/30 hover:to-teal-500/30
                       border border-cyan-500/30 hover:border-cyan-500/50
                       text-cyan-300 hover:text-cyan-200
                       disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Download className="w-4 h-4" />
            {t('export.quick.save')}
          </button>
        </Tooltip>

        {/* Library */}
        <Tooltip text={t('tip.library')}>
          <button
            onClick={onOpenLibrary}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg font-medium transition-all duration-200
                       bg-gradient-to-r from-amber-500/20 to-orange-500/20 hover:from-amber-500/30 hover:to-orange-500/30
                       border border-amber-500/30 hover:border-amber-500/50
                       text-amber-300 hover:text-amber-200"
          >
            <Image className="w-4 h-4" />
            {t('export.library')}
          </button>
        </Tooltip>

        {/* Save As */}
        <Tooltip text={t('tip.save.as')}>
          <button
            onClick={onSave}
            disabled={isExporting}
            className="flex items-center gap-1.5 px-4 py-1.5 text-sm rounded-lg font-medium transition-all duration-200
                       bg-gradient-to-r from-violet-500 to-purple-600 hover:from-violet-400 hover:to-purple-500
                       text-white shadow-lg shadow-violet-500/25 hover:shadow-violet-500/40
                       hover:-translate-y-0.5 active:translate-y-0
                       disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0"
          >
            <Save className="w-4 h-4" />
            {t('export.save.as')}
          </button>
        </Tooltip>
      </div>

      {/* Exporting indicator */}
      {isExporting && (
        <div className="flex items-center gap-2 ml-2">
          <div className="w-2 h-2 rounded-full bg-violet-400 animate-pulse" />
          <span className="text-sm text-violet-300 animate-pulse font-medium">{t('export.exporting')}</span>
        </div>
      )}
    </div>
  );
}
