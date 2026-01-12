import { useState } from 'react'
import {
  Settings,
  Plus,
  Save,
  Trash2,
  X,
  Moon,
  Sun,
  Monitor,
  Laptop,
} from 'lucide-react'
import {
  useSettings,
  useUpdateSetting,
  useDeleteSetting,
} from '@/hooks/queries'
import { useTheme } from '@/components/theme-provider'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  Button,
  Input,
} from '@/components/ui'
import { cn } from '@/lib/utils'
import { PageHeader } from '@/components/layout/page-header'

type Theme = 'light' | 'dark' | 'system'

export function SettingsPage() {
  return (
    <div className="flex flex-col h-full bg-background">
      <PageHeader
        icon={Settings}
        iconClassName="text-zinc-500"
        title="Settings"
        description="Configure your maxx-next instance"
      />

      <div className="flex-1 overflow-y-auto p-6">
        <div className="space-y-6">
          <AppearanceSection />
          <SystemSettingsSection />
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

function SystemSettingsSection() {
  const { data: settings, isLoading } = useSettings()
  const updateSetting = useUpdateSetting()
  const deleteSetting = useDeleteSetting()
  const [newKey, setNewKey] = useState('')
  const [newValue, setNewValue] = useState('')
  const [showAddForm, setShowAddForm] = useState(false)
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')

  const handleAdd = () => {
    if (!newKey.trim()) return
    updateSetting.mutate(
      { key: newKey.trim(), value: newValue },
      {
        onSuccess: () => {
          setNewKey('')
          setNewValue('')
          setShowAddForm(false)
        },
      }
    )
  }

  const handleEdit = (key: string) => {
    if (!settings) return
    setEditingKey(key)
    setEditValue(settings[key] || '')
  }

  const handleSave = (key: string) => {
    updateSetting.mutate(
      { key, value: editValue },
      {
        onSuccess: () => {
          setEditingKey(null)
          setEditValue('')
        },
      }
    )
  }

  const handleDelete = (key: string) => {
    if (confirm(`Delete setting "${key}"?`)) {
      deleteSetting.mutate(key)
    }
  }

  const settingsEntries = settings ? Object.entries(settings) : []

  return (
    <Card className="border-border bg-surface-primary">
      <CardHeader className="border-b border-border py-4 flex flex-row items-center justify-between space-y-0">
        <CardTitle className="text-base font-medium flex items-center gap-2">
          <Settings className="h-4 w-4 text-text-muted" />
          System Configuration
        </CardTitle>
        <Button
          onClick={() => setShowAddForm(true)}
          size="sm"
          className={cn(
            'h-8 gap-1.5',
            showAddForm && 'opacity-50 pointer-events-none'
          )}
          disabled={showAddForm}
        >
          <Plus size={14} />
          Add Setting
        </Button>
      </CardHeader>

      <CardContent className="p-0">
        {isLoading ? (
          <div className="p-8 text-center text-text-muted flex items-center justify-center gap-2">
            <div className="animate-spin h-4 w-4 border-2 border-text-muted border-t-transparent rounded-full" />
            Loading settings...
          </div>
        ) : settingsEntries.length === 0 && !showAddForm ? (
          <div className="p-12 text-center text-text-muted">
            <div className="w-12 h-12 rounded-full bg-surface-secondary flex items-center justify-center mx-auto mb-3">
              <Settings size={20} className="opacity-30" />
            </div>
            <p className="text-sm">No custom settings configured</p>
          </div>
        ) : (
          <div className="divide-y divide-border">
            {/* Header Row for large screens */}
            <div className="hidden md:flex bg-surface-secondary/30 px-6 py-2 text-xs font-medium text-text-secondary uppercase tracking-wider">
              <div className="w-1/3">Key</div>
              <div className="flex-1">Value</div>
              <div className="w-20 text-right">Actions</div>
            </div>

            {showAddForm && (
              <div className="p-4 bg-accent/5 animate-in slide-in-from-top-2 duration-200">
                <div className="flex flex-col md:flex-row gap-3 items-start md:items-center">
                  <div className="w-full md:w-1/3">
                    <Input
                      type="text"
                      value={newKey}
                      onChange={e => setNewKey(e.target.value)}
                      placeholder="Key (e.g., system.timeout)"
                      className="bg-surface-primary border-accent/30 focus:border-accent font-mono text-sm"
                      autoFocus
                    />
                  </div>
                  <div className="w-full md:flex-1">
                    <Input
                      type="text"
                      value={newValue}
                      onChange={e => setNewValue(e.target.value)}
                      placeholder="Value"
                      className="bg-surface-primary border-accent/30 focus:border-accent font-mono text-sm"
                    />
                  </div>
                  <div className="flex items-center gap-2 md:w-20 md:justify-end">
                    <Button
                      onClick={handleAdd}
                      disabled={!newKey.trim() || updateSetting.isPending}
                      size="sm"
                      className="h-8 w-8 p-0 bg-accent text-white hover:bg-accent-hover"
                    >
                      <Save size={14} />
                    </Button>
                    <Button
                      onClick={() => {
                        setShowAddForm(false)
                        setNewKey('')
                        setNewValue('')
                      }}
                      size="sm"
                      variant="ghost"
                      className="h-8 w-8 p-0 hover:bg-error/10 hover:text-error"
                    >
                      <X size={14} />
                    </Button>
                  </div>
                </div>
              </div>
            )}

            {settingsEntries.map(([key, value]) => (
              <div
                key={key}
                className="p-4 md:px-6 flex flex-col md:flex-row md:items-center gap-2 md:gap-4 group transition-colors hover:bg-surface-secondary/30"
              >
                {/* Key Column */}
                <div className="w-full md:w-1/3 min-w-0">
                  <div
                    className="text-xs md:text-sm font-mono font-medium text-text-secondary md:text-text-primary break-all"
                    title={key}
                  >
                    {key}
                  </div>
                </div>

                {/* Value Column */}
                <div className="w-full md:flex-1 min-w-0">
                  {editingKey === key ? (
                    <Input
                      type="text"
                      value={editValue}
                      onChange={e => setEditValue(e.target.value)}
                      className="h-8 text-sm font-mono bg-surface-primary w-full"
                      autoFocus
                      onKeyDown={e => {
                        if (e.key === 'Enter') handleSave(key)
                        if (e.key === 'Escape') setEditingKey(null)
                      }}
                    />
                  ) : (
                    <div
                      className="text-sm text-text-primary font-mono break-all cursor-pointer hover:text-accent transition-colors py-1"
                      onClick={() => handleEdit(key)}
                      title="Click to edit"
                    >
                      {value || (
                        <span className="italic text-text-muted opacity-50">
                          (empty)
                        </span>
                      )}
                    </div>
                  )}
                </div>

                {/* Actions Column */}
                <div className="flex items-center justify-end gap-2 md:w-20 opacity-100 md:opacity-0 md:group-hover:opacity-100 transition-opacity">
                  {editingKey === key ? (
                    <>
                      <Button
                        onClick={() => handleSave(key)}
                        disabled={updateSetting.isPending}
                        size="sm"
                        className="h-8 w-8 p-0"
                      >
                        <Save size={14} />
                      </Button>
                      <Button
                        onClick={() => setEditingKey(null)}
                        size="sm"
                        variant="ghost"
                        className="h-8 w-8 p-0"
                      >
                        <X size={14} />
                      </Button>
                    </>
                  ) : (
                    <>
                      <Button
                        onClick={() => handleEdit(key)}
                        size="sm"
                        variant="ghost"
                        className="h-8 w-8 p-0 text-text-secondary hover:text-primary"
                      >
                        <Settings size={14} />
                      </Button>
                      <Button
                        onClick={() => handleDelete(key)}
                        size="sm"
                        variant="ghost"
                      >
                        <Trash2 size={14} />
                      </Button>
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default SettingsPage
