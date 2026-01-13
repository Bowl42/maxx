import {
  Server,
  Wand2,
  ChevronLeft,
  Layers,
  Grid3X3,
  CheckCircle2,
  FilePlus,
} from 'lucide-react'
import {
  quickTemplates,
  type ProviderFormData,
} from '../types'
import { Button } from '@/components/ui'

interface SelectTypeStepProps {
  formData: ProviderFormData
  onSelectType: (type: 'custom' | 'antigravity') => void
  onApplyTemplate: (templateId: string) => void
  onSkipToConfig: () => void
  onBack: () => void
}

export function SelectTypeStep({
  formData,
  onSelectType,
  onApplyTemplate,
  onSkipToConfig,
  onBack,
}: SelectTypeStepProps) {
  return (
    <div className="flex flex-col h-full">
      <div className="px-6 h-[73px] flex items-center gap-4  border-b border-border bg-card">
        <Button onClick={onBack} variant={'ghost'}>
          <ChevronLeft size={20} />
        </Button>
        <div>
          <h2 className="text-headline font-semibold text-foreground">
            Add Provider
          </h2>
          <p className="text-caption text-muted-foreground">
            Choose a service provider to get started
          </p>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <div className="container mx-auto max-w-[1600px] space-y-10">
          {/* Section: Service Provider */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-foreground border-b border-border pb-2">
              1. Choose Service Provider
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Button
                onClick={() => onSelectType('antigravity')}
                variant="ghost"
                className={`group p-0 rounded-lg border text-left transition-all h-auto ${
                  formData.type === 'antigravity'
                    ? 'border-provider-antigravity bg-provider-antigravity/10'
                    : 'border-border bg-card hover:bg-muted'
                }`}
              >
                <div className="p-5 flex items-center gap-4">
                  <div className="w-12 h-12 rounded-md bg-provider-antigravity/15 flex items-center justify-center shrink-0">
                    <Wand2 size={24} className="text-provider-antigravity" />
                  </div>

                  <div className="flex-1 min-w-0">
                    <h3 className="text-headline font-semibold text-foreground mb-1">
                      Antigravity Cloud
                    </h3>
                    <p className="text-caption text-muted-foreground">
                      Zero-config managed service with OAuth
                    </p>
                  </div>

                  {formData.type === 'antigravity' && (
                    <CheckCircle2 size={20} className="text-provider-antigravity shrink-0" />
                  )}
                </div>
              </Button>

              <Button
                onClick={() => onSelectType('custom')}
                variant="ghost"
                className={`group p-0 rounded-lg border text-left transition-all h-auto ${
                  formData.type === 'custom'
                    ? 'border-provider-custom bg-provider-custom/10'
                    : 'border-border bg-card hover:bg-muted'
                }`}
              >
                <div className="p-5 flex items-center gap-4">
                  <div className="w-12 h-12 rounded-md bg-provider-custom/15 flex items-center justify-center shrink-0">
                    <Server size={24} className="text-provider-custom" />
                  </div>

                  <div className="flex-1 min-w-0">
                    <h3 className="text-headline font-semibold text-foreground mb-1">
                      Custom Provider
                    </h3>
                    <p className="text-caption text-muted-foreground">
                      Configure your own API endpoint
                    </p>
                  </div>

                  {formData.type === 'custom' && (
                    <CheckCircle2 size={20} className="text-provider-custom shrink-0" />
                  )}
                </div>
              </Button>
            </div>
          </div>

          {/* Section: Templates (Custom only) */}
          {formData.type === 'custom' && (
            <div className="space-y-4 animate-in fade-in slide-in-from-bottom-4 duration-300">
              <div className="flex items-center justify-between border-b border-border pb-2">
                <h3 className="text-lg font-semibold text-foreground">
                  2. Select a Template{' '}
                  <span className="text-muted-foreground font-normal text-sm ml-2">
                    (Optional)
                  </span>
                </h3>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                {/* Empty Template Card */}
                <Button
                  onClick={onSkipToConfig}
                  variant="ghost"
                  className="group p-0 rounded-lg border border-dashed border-border bg-card hover:bg-muted transition-all h-auto"
                >
                  <div className="p-4 flex flex-col gap-3">
                    <div className="w-10 h-10 rounded-md bg-muted flex items-center justify-center">
                      <FilePlus size={20} className="text-muted-foreground group-hover:text-primary transition-colors" />
                    </div>

                    <div>
                      <h4 className="text-body font-semibold text-foreground mb-0.5">
                        Empty Template
                      </h4>
                      <p className="text-caption text-muted-foreground">
                        Start from scratch
                      </p>
                    </div>
                  </div>
                </Button>

                {quickTemplates.map(template => {
                  const Icon = template.icon === 'grid' ? Grid3X3 : Layers
                  const isSelected = formData.selectedTemplate === template.id
                  return (
                    <Button
                      key={template.id}
                      onClick={() => onApplyTemplate(template.id)}
                      variant="ghost"
                      className={`group p-0 rounded-lg border text-left transition-all h-auto ${
                        isSelected
                          ? 'border-primary bg-primary/10'
                          : 'border-border bg-card hover:bg-muted'
                      }`}
                    >
                      <div className="p-4 flex flex-col gap-3">
                        <div className="flex items-center justify-between">
                          <div
                            className={`w-10 h-10 rounded-md flex items-center justify-center transition-colors ${
                              isSelected
                                ? 'bg-primary/15'
                                : 'bg-muted group-hover:bg-primary/10'
                            }`}
                          >
                            <Icon
                              size={20}
                              className={
                                isSelected
                                  ? 'text-primary'
                                  : 'text-muted-foreground group-hover:text-primary transition-colors'
                              }
                            />
                          </div>
                          {isSelected && (
                            <CheckCircle2 size={18} className="text-primary" />
                          )}
                        </div>

                        <div>
                          <h4
                            className={`text-body font-semibold mb-0.5 transition-colors ${
                              isSelected ? 'text-primary' : 'text-foreground'
                            }`}
                          >
                            {template.name}
                          </h4>
                          <p className="text-caption text-muted-foreground">
                            {template.description}
                          </p>
                        </div>
                      </div>
                    </Button>
                  )
                })}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
