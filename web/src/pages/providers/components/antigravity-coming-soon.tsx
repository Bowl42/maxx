import { Wand2, ChevronLeft } from 'lucide-react'
import { ANTIGRAVITY_COLOR } from '../types'

interface AntigravityComingSoonProps {
  onBack: () => void
}

export function AntigravityComingSoon({ onBack }: AntigravityComingSoonProps) {
  return (
    <div className="flex flex-col h-full">
      <div className="h-[73px] flex items-center gap-4 p-lg border-b border-border bg-surface-primary">
        <button
          onClick={onBack}
          className="p-1.5 -ml-1 rounded-lg hover:bg-surface-hover text-text-secondary hover:text-text-primary transition-colors"
        >
          <ChevronLeft size={20} />
        </button>
        <div>
          <h2 className="text-headline font-semibold text-text-primary">
            Antigravity
          </h2>
          <p className="text-caption text-text-secondary">
            OAuth Authentication
          </p>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-lg">
        <div className="container mx-auto max-w-[1600px]">
          <div className="flex flex-col items-center justify-center py-16">
            <div
              className="w-20 h-20 rounded-2xl flex items-center justify-center mb-6"
              style={{ backgroundColor: `${ANTIGRAVITY_COLOR}15` }}
            >
              <Wand2 size={40} style={{ color: ANTIGRAVITY_COLOR }} />
            </div>
            <h3 className="text-title3 font-semibold text-text-primary mb-2">
              Coming Soon
            </h3>
            <p className="text-body text-text-secondary text-center max-w-sm">
              Antigravity OAuth authentication will be available in a future
              update.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
