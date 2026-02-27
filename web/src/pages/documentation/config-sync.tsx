import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Wrench, FileText, AlertTriangle, CheckCircle2 } from 'lucide-react';
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui';
import { PageHeader } from '@/components/layout/page-header';
import {
  useAPITokens,
  useProjects,
  useProxyStatus,
  useResponseModels,
  useSyncCodexLocalConfig,
} from '@/hooks/queries';
import type { CodexLocalConfigSyncResult } from '@/lib/transport';

const GLOBAL_ROUTE_VALUE = 'global';
const FALLBACK_CODEX_MODELS = ['gpt-5.3-codex', 'gpt-5.2-codex', 'gpt-5.1-codex'];

function buildBaseUrl(address: string): string {
  const trimmedAddress = address.trim().replace(/\/+$/, '');
  if (!trimmedAddress) {
    return '';
  }

  if (/^https?:\/\//i.test(trimmedAddress)) {
    return trimmedAddress;
  }

  const protocol =
    typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'https' : 'http';
  return `${protocol}://${trimmedAddress}`;
}

function buildRouteBaseUrl(baseUrl: string, projectSlug: string | undefined): string {
  const normalizedBaseUrl = baseUrl.trim().replace(/\/+$/, '');
  if (!normalizedBaseUrl) {
    return '';
  }
  if (!projectSlug) {
    return normalizedBaseUrl;
  }

  const normalizedSlug = projectSlug.trim().replace(/^\/+/, '').replace(/\/+$/, '');
  if (!normalizedSlug) {
    return normalizedBaseUrl;
  }

  return `${normalizedBaseUrl}/project/${normalizedSlug}`;
}

function buildRouteValue(projectSlug: string | undefined): string {
  if (!projectSlug) {
    return GLOBAL_ROUTE_VALUE;
  }
  return `project:${projectSlug}`;
}

function extractProjectSlug(routeValue: string): string | undefined {
  if (!routeValue.startsWith('project:')) {
    return undefined;
  }

  const projectSlug = routeValue.slice('project:'.length).trim();
  return projectSlug || undefined;
}

function extractErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null) {
    const responseData = (error as { response?: { data?: { error?: string } } }).response?.data;
    if (responseData && typeof responseData.error === 'string' && responseData.error.trim() !== '') {
      return responseData.error;
    }

    const message = (error as { message?: string }).message;
    if (typeof message === 'string' && message.trim() !== '') {
      return message;
    }
  }

  return fallback;
}

function buildModelOptions(responseModels: string[] | undefined): string[] {
  const models = responseModels ?? [];
  const codexLike = models.filter((model) => /codex/i.test(model));
  const gpt5Like = models.filter((model) => /^gpt-5(\.|-|$)/i.test(model));

  const seen = new Set<string>();
  const ordered = [...codexLike, ...gpt5Like, ...models, ...FALLBACK_CODEX_MODELS];
  return ordered.filter((model) => {
    const key = model.trim();
    if (!key || seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
}

export function DocumentationConfigPage() {
  const { t } = useTranslation();
  const { data: proxyStatus } = useProxyStatus();
  const { data: projects, isLoading: isLoadingProjects } = useProjects();
  const { data: apiTokens, isLoading: isLoadingTokens } = useAPITokens();
  const { data: responseModels, isLoading: isLoadingModels } = useResponseModels();
  const syncCodexLocalConfig = useSyncCodexLocalConfig();

  const [selectedTokenId, setSelectedTokenId] = useState('');
  const [selectedRouteValue, setSelectedRouteValue] = useState(GLOBAL_ROUTE_VALUE);
  const [selectedModel, setSelectedModel] = useState('');
  const [syncResult, setSyncResult] = useState<CodexLocalConfigSyncResult | null>(null);
  const [syncError, setSyncError] = useState('');

  const baseUrl = useMemo(() => {
    const proxyAddress = proxyStatus?.address ?? 'localhost:9880';
    return buildBaseUrl(proxyAddress);
  }, [proxyStatus?.address]);

  const selectedToken = useMemo(
    () => apiTokens?.find((token) => String(token.id) === selectedTokenId),
    [apiTokens, selectedTokenId],
  );
  const selectedProjectSlug = useMemo(
    () => extractProjectSlug(selectedRouteValue),
    [selectedRouteValue],
  );
  const selectedProject = useMemo(() => {
    if (!selectedProjectSlug) {
      return null;
    }
    return (projects ?? []).find((project) => project.slug === selectedProjectSlug) ?? null;
  }, [projects, selectedProjectSlug]);
  const routeDisplayText = selectedProject
    ? `${selectedProject.name} (${selectedProject.slug})`
    : t('documentationConfig.routeGlobal');
  const routeBaseUrl = useMemo(
    () => buildRouteBaseUrl(baseUrl, selectedProjectSlug),
    [baseUrl, selectedProjectSlug],
  );
  const tokenDisplayText = selectedToken
    ? `${selectedToken.name} (${selectedToken.tokenPrefix})`
    : t('documentationConfig.tokenPlaceholder');
  const modelDisplayText = selectedModel || t('documentationConfig.modelPlaceholder');
  const modelOptions = useMemo(() => buildModelOptions(responseModels), [responseModels]);

  useEffect(() => {
    const projectSlug = extractProjectSlug(selectedRouteValue);
    if (!projectSlug) {
      return;
    }
    const hasProject = (projects ?? []).some((project) => project.slug === projectSlug);
    if (!hasProject) {
      setSelectedRouteValue(GLOBAL_ROUTE_VALUE);
    }
  }, [projects, selectedRouteValue]);

  useEffect(() => {
    if (modelOptions.length === 0) {
      if (selectedModel !== '') {
        setSelectedModel('');
      }
      return;
    }

    if (!selectedModel || !modelOptions.includes(selectedModel)) {
      setSelectedModel(modelOptions[0]);
    }
  }, [modelOptions, selectedModel]);

  const handleSync = async () => {
    if (!selectedToken) {
      return;
    }

    setSyncError('');
    setSyncResult(null);

    try {
      const result = await syncCodexLocalConfig.mutateAsync({
        apiToken: selectedToken.token,
        providerName: 'maxx',
        projectSlug: selectedProjectSlug,
        model: selectedModel || undefined,
      });
      setSyncResult(result);
    } catch (error) {
      setSyncError(extractErrorMessage(error, t('documentationConfig.syncFailed')));
    }
  };

  const hasTokens = !!apiTokens && apiTokens.length > 0;

  return (
    <div className="flex flex-col h-full">
      <PageHeader
        icon={Wrench}
        iconClassName="text-emerald-500"
        title={t('documentationConfig.title')}
        description={t('documentationConfig.description')}
      />

      <div className="flex-1 overflow-y-auto p-4 md:p-6">
        <div className="max-w-4xl mx-auto space-y-6">
          <Card className="border-border bg-card">
            <CardHeader className="border-b border-border">
              <CardTitle className="text-base font-medium flex items-center gap-2">
                <FileText className="h-4 w-4 text-muted-foreground" />
                {t('documentationConfig.codexConfigTitle')}
              </CardTitle>
              <CardDescription>{t('documentationConfig.codexConfigDesc')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-5 pt-6">
              <div className="space-y-2">
                <p className="text-sm font-medium">{t('documentationConfig.routeLabel')}</p>
                <Select
                  value={selectedRouteValue}
                  onValueChange={(value) => setSelectedRouteValue(value ?? GLOBAL_ROUTE_VALUE)}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue>{routeDisplayText}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={GLOBAL_ROUTE_VALUE}>
                      {t('documentationConfig.routeGlobal')}
                    </SelectItem>
                    {(projects ?? []).map((project) => (
                      <SelectItem key={project.id} value={buildRouteValue(project.slug)}>
                        {project.name} ({project.slug})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {!isLoadingProjects && (!projects || projects.length === 0) && (
                  <p className="text-xs text-muted-foreground">
                    {t('documentationConfig.routeNoProjectsHint')}
                  </p>
                )}
              </div>

              <div className="space-y-2">
                <p className="text-sm font-medium">{t('documentationConfig.baseUrl')}</p>
                <div className="rounded-md border border-border bg-muted/40 px-3 py-2">
                  <code className="text-xs break-all">{routeBaseUrl}</code>
                </div>
              </div>

              <div className="space-y-2">
                <p className="text-sm font-medium">{t('documentationConfig.tokenLabel')}</p>
                <Select
                  value={selectedTokenId}
                  onValueChange={(value) => setSelectedTokenId(value ?? '')}
                  disabled={!hasTokens}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue>{tokenDisplayText}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {(apiTokens ?? []).map((token) => (
                      <SelectItem key={token.id} value={String(token.id)}>
                        {token.name} ({token.tokenPrefix})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>

                {!isLoadingTokens && !hasTokens && (
                  <p className="text-xs text-amber-600 dark:text-amber-400">
                    {t('documentationConfig.noTokens')}{' '}
                    <Link to="/api-tokens" className="underline hover:opacity-80">
                      {t('documentationConfig.goToApiTokens')}
                    </Link>
                  </p>
                )}
              </div>

              <div className="space-y-2">
                <p className="text-sm font-medium">{t('documentationConfig.modelLabel')}</p>
                <Select value={selectedModel} onValueChange={(value) => setSelectedModel(value ?? '')}>
                  <SelectTrigger className="w-full">
                    <SelectValue>{modelDisplayText}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    {modelOptions.map((model) => (
                      <SelectItem key={model} value={model}>
                        {model}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>

                {!isLoadingModels && (!responseModels || responseModels.length === 0) && (
                  <p className="text-xs text-muted-foreground">
                    {t('documentationConfig.noModelsHint')}
                  </p>
                )}
              </div>

              <div className="space-y-2">
                <p className="text-sm font-medium">{t('documentationConfig.targetFiles')}</p>
                <ul className="space-y-1 text-xs text-muted-foreground">
                  <li>~/.codex/config.toml</li>
                  <li>~/.codex/auth.json</li>
                </ul>
              </div>

              <Button
                onClick={handleSync}
                disabled={!selectedToken || syncCodexLocalConfig.isPending}
                className="w-full sm:w-auto"
              >
                {syncCodexLocalConfig.isPending
                  ? t('documentationConfig.syncing')
                  : t('documentationConfig.syncNow')}
              </Button>
            </CardContent>
          </Card>

          {syncResult && (
            <Card className="border-border bg-card">
              <CardContent className="space-y-3 pt-6">
                <div className="flex items-center gap-2 text-emerald-600 dark:text-emerald-400">
                  <CheckCircle2 className="h-4 w-4" />
                  <p className="text-sm font-medium">{t('documentationConfig.syncSuccess')}</p>
                </div>
                <p className="text-xs text-muted-foreground">
                  {t('documentationConfig.syncSuccessDesc')}
                </p>
                <div className="rounded-md border border-border bg-muted/40 px-3 py-2">
                  <p className="text-xs font-medium mb-2">{t('documentationConfig.writtenFiles')}</p>
                  <ul className="space-y-1 text-xs text-muted-foreground">
                    {syncResult.writtenFiles.map((filePath) => (
                      <li key={filePath} className="break-all">
                        {filePath}
                      </li>
                    ))}
                  </ul>
                </div>

                {syncResult.recoveredAuthJSON && (
                  <div className="rounded-md border border-amber-500/30 bg-amber-500/10 px-3 py-2">
                    <p className="text-xs text-amber-700 dark:text-amber-300">
                      {t('documentationConfig.recoveredAuthNotice')}
                    </p>
                    {syncResult.backupFile && (
                      <p className="mt-1 text-xs text-amber-700 dark:text-amber-300 break-all">
                        {syncResult.backupFile}
                      </p>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {syncError && (
            <div className="flex items-start gap-2 rounded-md border border-destructive/30 bg-destructive/10 p-3">
              <AlertTriangle className="h-4 w-4 text-destructive mt-0.5 shrink-0" />
              <p className="text-sm text-destructive">{syncError}</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default DocumentationConfigPage;
