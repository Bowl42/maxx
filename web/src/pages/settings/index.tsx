import { Settings, Moon, Sun, Monitor, Laptop, FolderOpen } from 'lucide-react'
import { useTheme } from '@/components/theme-provider'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Button,
  Input,
  Switch,
} from '@/components/ui'
import { PageHeader } from '@/components/layout/page-header'
import { useSettings, useUpdateSetting } from '@/hooks/queries'

type Theme = 'light' | 'dark' | 'system'

export function SettingsPage() {
  return (
    <div className="flex flex-col h-full bg-background">
      <PageHeader
        icon={Settings}
        iconClassName="text-zinc-500"
        title="Settings"
        description="Configure your maxx instance"
      />

      <div className="flex-1 overflow-y-auto p-6">
        <div className="space-y-6">
          <AppearanceSection />
          <ForceProjectSection />
        </div>
      </div>
    </div>
  )
}

function AppearanceSection() {
  const { theme, setTheme } = useTheme()

  const themes: { value: Theme; label: string; icon: typeof Sun }[] = [
    { value: 'light', label: 'Light', icon: Sun },
    { value: 'dark', label: 'Dark', icon: Moon },
    { value: 'system', label: 'System', icon: Laptop },
  ]

  return (
    <Card className="border-border bg-surface-primary">
      <CardHeader className="border-b border-border py-4">
        <CardTitle className="text-base font-medium flex items-center gap-2">
          <Monitor className="h-4 w-4 text-text-muted" />
          Appearance
        </CardTitle>
      </CardHeader>
      <CardContent className="p-6">
        <div className="flex items-center gap-6">
          <label className="text-sm font-medium text-text-secondary w-40 shrink-0">
            Theme Preference
          </label>
          <div className="flex flex-wrap gap-3">
            {themes.map(({ value, label, icon: Icon }) => (
              <Button
                key={value}
                onClick={() => setTheme(value)}
                variant={theme === value ? 'default' : 'outline'}
              >
                <Icon size={16} />
                <span className="text-sm font-medium">{label}</span>
              </Button>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function ForceProjectSection() {
  const { data: settings, isLoading } = useSettings()
  const updateSetting = useUpdateSetting()

  const forceProjectEnabled = settings?.force_project_binding === 'true'
  const timeout = settings?.force_project_timeout || '30'

  const handleToggle = async (checked: boolean) => {
    await updateSetting.mutateAsync({
      key: 'force_project_binding',
      value: checked ? 'true' : 'false',
    })
  }

  const handleTimeoutChange = async (value: string) => {
    const numValue = parseInt(value, 10)
    if (numValue >= 5 && numValue <= 300) {
      await updateSetting.mutateAsync({
        key: 'force_project_timeout',
        value: value,
      })
    }
  }

  if (isLoading) return null

  return (
    <Card className="border-border bg-surface-primary">
      <CardHeader className="border-b border-border py-4">
        <CardTitle className="text-base font-medium flex items-center gap-2">
          <FolderOpen className="h-4 w-4 text-text-muted" />
          强制项目绑定
        </CardTitle>
      </CardHeader>
      <CardContent className="p-6 space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <label className="text-sm font-medium text-text-primary">
              启用强制项目绑定
            </label>
            <p className="text-xs text-text-muted mt-1">
              开启后，新会话必须选择项目才能继续执行请求
            </p>
          </div>
          <Switch
            checked={forceProjectEnabled}
            onCheckedChange={handleToggle}
            disabled={updateSetting.isPending}
          />
        </div>

        {forceProjectEnabled && (
          <div className="flex items-center gap-6 pt-4 border-t border-border">
            <label className="text-sm font-medium text-text-secondary w-32 shrink-0">
              等待超时（秒）
            </label>
            <Input
              type="number"
              value={timeout}
              onChange={e => handleTimeoutChange(e.target.value)}
              className="w-24"
              min={5}
              max={300}
              disabled={updateSetting.isPending}
            />
            <span className="text-xs text-text-muted">5 - 300 秒</span>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default SettingsPage
