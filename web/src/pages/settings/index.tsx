import { useState } from 'react';
import { Settings, Plus, Save, Trash2, X, Moon, Sun, Monitor, Laptop } from 'lucide-react';
import { useSettings, useUpdateSetting, useDeleteSetting } from '@/hooks/queries';
import { useTheme, type Theme } from '@/hooks/use-theme';
import { Card, CardContent, CardHeader, CardTitle, Button, Input } from '@/components/ui';
import { cn } from '@/lib/utils';

export function SettingsPage() {
  return (
    <div className="flex flex-col h-full bg-background">
      {/* Header */}
      <div className="h-[73px] flex items-center gap-3 px-6 border-b border-border bg-surface-primary flex-shrink-0">
        <div className="p-2 bg-accent/10 rounded-lg">
          <Settings size={20} className="text-accent" />
        </div>
        <div>
          <h1 className="text-lg font-semibold text-text-primary leading-tight">Settings</h1>
          <p className="text-xs text-text-secondary">Configure your maxx-next instance</p>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-4xl space-y-6">
          <AppearanceSection />
          <SystemSettingsSection />
        </div>
      </div>
    </div>
  );
}

function AppearanceSection() {
  const { theme, setTheme } = useTheme();

  const themes: { value: Theme; label: string; icon: typeof Sun }[] = [
    { value: 'light', label: 'Light', icon: Sun },
    { value: 'dark', label: 'Dark', icon: Moon },
    { value: 'system', label: 'System', icon: Laptop },
  ];

  return (
    <Card className="border-border bg-surface-primary">
      <CardHeader className="border-b border-border py-4">
        <CardTitle className="text-base font-medium flex items-center gap-2">
          <Monitor className="h-4 w-4 text-text-muted" />
          Appearance
        </CardTitle>
      </CardHeader>
      <CardContent className="p-6">
        <div className="space-y-3">
          <label className="text-sm font-medium text-text-secondary block">Theme Preference</label>
          <div className="flex flex-wrap gap-3">
            {themes.map(({ value, label, icon: Icon }) => (
              <button
                key={value}
                onClick={() => setTheme(value)}
                className={cn(
                  "flex items-center gap-2 px-4 py-2.5 rounded-lg border transition-all duration-200",
                  theme === value
                    ? "border-accent bg-accent/10 text-accent ring-1 ring-accent/20"
                    : "border-border bg-surface-secondary text-text-secondary hover:bg-surface-hover hover:border-text-muted/50"
                )}
              >
                <Icon size={16} />
                <span className="text-sm font-medium">{label}</span>
              </button>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function SystemSettingsSection() {
  const { data: settings, isLoading } = useSettings();
  const updateSetting = useUpdateSetting();
  const deleteSetting = useDeleteSetting();
  const [newKey, setNewKey] = useState('');
  const [newValue, setNewValue] = useState('');
  const [showAddForm, setShowAddForm] = useState(false);
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');

  const handleAdd = () => {
    if (!newKey.trim()) return;
    updateSetting.mutate(
      { key: newKey.trim(), value: newValue },
      {
        onSuccess: () => {
          setNewKey('');
          setNewValue('');
          setShowAddForm(false);
        },
      }
    );
  };

  const handleEdit = (key: string) => {
    if (!settings) return;
    setEditingKey(key);
    setEditValue(settings[key] || '');
  };

  const handleSave = (key: string) => {
    updateSetting.mutate(
      { key, value: editValue },
      {
        onSuccess: () => {
          setEditingKey(null);
          setEditValue('');
        },
      }
    );
  };

  const handleDelete = (key: string) => {
    if (confirm(`Delete setting "${key}"?`)) {
      deleteSetting.mutate(key);
    }
  };

  const settingsEntries = settings ? Object.entries(settings) : [];

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
          className={cn("h-8 gap-1.5", showAddForm && "opacity-50 pointer-events-none")}
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
            {showAddForm && (
              <div className="p-4 bg-accent/5 animate-in slide-in-from-top-2 duration-200">
                <div className="flex gap-3 items-start">
                  <div className="flex-1 space-y-2">
                    <Input
                      type="text"
                      value={newKey}
                      onChange={(e) => setNewKey(e.target.value)}
                      placeholder="Key (e.g., system.timeout)"
                      className="bg-surface-primary border-accent/30 focus:border-accent"
                      autoFocus
                    />
                    <Input
                      type="text"
                      value={newValue}
                      onChange={(e) => setNewValue(e.target.value)}
                      placeholder="Value"
                      className="bg-surface-primary border-accent/30 focus:border-accent"
                    />
                  </div>
                  <div className="flex flex-col gap-2 pt-0.5">
                    <Button
                      onClick={handleAdd}
                      disabled={!newKey.trim() || updateSetting.isPending}
                      size="sm"
                      className="h-9 w-9 p-0 bg-accent text-white hover:bg-accent-hover"
                    >
                      <Save size={14} />
                    </Button>
                    <Button
                      onClick={() => {
                        setShowAddForm(false);
                        setNewKey('');
                        setNewValue('');
                      }}
                      size="sm"
                      variant="ghost"
                      className="h-9 w-9 p-0 hover:bg-error/10 hover:text-error"
                    >
                      <X size={14} />
                    </Button>
                  </div>
                </div>
              </div>
            )}
            
            {settingsEntries.map(([key, value]) => (
              <div key={key} className="p-4 flex items-center gap-4 group transition-colors hover:bg-surface-secondary/30">
                <div className="flex-1 min-w-0">
                  <div className="text-xs font-mono font-medium text-text-secondary mb-1">{key}</div>
                  {editingKey === key ? (
                    <Input
                      type="text"
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      className="h-8 text-sm font-mono bg-surface-primary"
                      autoFocus
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') handleSave(key);
                        if (e.key === 'Escape') setEditingKey(null);
                      }}
                    />
                  ) : (
                    <div
                      className="text-sm text-text-primary font-mono truncate cursor-pointer hover:text-accent transition-colors py-1"
                      onClick={() => handleEdit(key)}
                      title="Click to edit"
                    >
                      {value || <span className="italic text-text-muted opacity-50">(empty)</span>}
                    </div>
                  )}
                </div>
                <div className="flex items-center gap-2 opacity-0 group-hover:opacity-100 transition-opacity focus-within:opacity-100">
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
                        className="h-8 w-8 p-0 text-text-secondary hover:text-error hover:bg-error/10"
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
  );
}

export default SettingsPage;
