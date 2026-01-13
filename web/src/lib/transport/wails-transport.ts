/**
 * Wails Transport 实现
 * 使用 Wails 自动生成的绑定调用 Go 方法
 */

import type { Transport, TransportConfig } from './interface';
import type {
  Provider,
  CreateProviderData,
  Project,
  CreateProjectData,
  Session,
  Route,
  CreateRouteData,
  RetryConfig,
  CreateRetryConfigData,
  RoutingStrategy,
  CreateRoutingStrategyData,
  ProxyRequest,
  ProxyUpstreamAttempt,
  ProxyStatus,
  ProviderStats,
  CursorPaginationParams,
  CursorPaginationResult,
  WSMessageType,
  EventCallback,
  UnsubscribeFn,
  AntigravityTokenValidationResult,
  AntigravityBatchValidationResult,
  AntigravityQuotaData,
  Cooldown,
  ImportResult,
} from './types';

// 导入 Wails 自动生成的绑定
import * as DesktopApp from '@/wailsjs/go/desktop/DesktopApp';

// Wails 事件 API 类型
type WailsEventCallback = (data: unknown) => void;
type WailsUnsubscribeFn = () => void;

// Wails v2 runtime 类型
declare global {
  interface Window {
    runtime?: {
      EventsOn: (eventName: string, callback: WailsEventCallback) => WailsUnsubscribeFn;
      EventsOff: (eventName: string) => void;
    };
  }
}

export class WailsTransport implements Transport {
  private connected = false;
  private eventUnsubscribers: Map<string, WailsUnsubscribeFn> = new Map();
  private eventCallbacks: Map<WSMessageType, Set<EventCallback>> = new Map();

  constructor(_config: TransportConfig = {}) {
    // Wails 模式下配置通常不需要
  }

  // ===== Provider API =====

  async getProviders(): Promise<Provider[]> {
    return DesktopApp.GetProviders() as Promise<Provider[]>;
  }

  async getProvider(id: number): Promise<Provider> {
    return DesktopApp.GetProvider(id) as Promise<Provider>;
  }

  async createProvider(data: CreateProviderData): Promise<Provider> {
    await DesktopApp.CreateProvider(data as any);
    // Wails CreateProvider returns void, need to get the created provider
    const providers = await this.getProviders();
    return providers[providers.length - 1];
  }

  async updateProvider(id: number, data: Partial<Provider>): Promise<Provider> {
    await DesktopApp.UpdateProvider({ ...data, id } as any);
    return this.getProvider(id);
  }

  async deleteProvider(id: number): Promise<void> {
    await DesktopApp.DeleteProvider(id);
  }

  async exportProviders(): Promise<Provider[]> {
    return DesktopApp.ExportProviders() as Promise<Provider[]>;
  }

  async importProviders(providers: Provider[]): Promise<ImportResult> {
    return DesktopApp.ImportProviders(providers as any) as Promise<ImportResult>;
  }

  // ===== Project API =====

  async getProjects(): Promise<Project[]> {
    return DesktopApp.GetProjects() as Promise<Project[]>;
  }

  async getProject(id: number): Promise<Project> {
    return DesktopApp.GetProject(id) as Promise<Project>;
  }

  async getProjectBySlug(slug: string): Promise<Project> {
    return DesktopApp.GetProjectBySlug(slug) as Promise<Project>;
  }

  async createProject(data: CreateProjectData): Promise<Project> {
    await DesktopApp.CreateProject(data as any);
    const projects = await this.getProjects();
    return projects[projects.length - 1];
  }

  async updateProject(id: number, data: Partial<Project>): Promise<Project> {
    await DesktopApp.UpdateProject({ ...data, id } as any);
    return this.getProject(id);
  }

  async deleteProject(id: number): Promise<void> {
    await DesktopApp.DeleteProject(id);
  }

  // ===== Route API =====

  async getRoutes(): Promise<Route[]> {
    return DesktopApp.GetRoutes() as Promise<Route[]>;
  }

  async getRoute(id: number): Promise<Route> {
    return DesktopApp.GetRoute(id) as Promise<Route>;
  }

  async createRoute(data: CreateRouteData): Promise<Route> {
    await DesktopApp.CreateRoute(data as any);
    const routes = await this.getRoutes();
    return routes[routes.length - 1];
  }

  async updateRoute(id: number, data: Partial<Route>): Promise<Route> {
    await DesktopApp.UpdateRoute({ ...data, id } as any);
    return this.getRoute(id);
  }

  async deleteRoute(id: number): Promise<void> {
    await DesktopApp.DeleteRoute(id);
  }

  // ===== Session API =====

  async getSessions(): Promise<Session[]> {
    return DesktopApp.GetSessions() as Promise<Session[]>;
  }

  async updateSessionProject(
    sessionID: string,
    projectID: number
  ): Promise<{ session: Session; updatedRequests: number }> {
    const result = await DesktopApp.UpdateSessionProject(sessionID, projectID);
    return result as { session: Session; updatedRequests: number };
  }

  async rejectSession(sessionID: string): Promise<Session> {
    return DesktopApp.RejectSession(sessionID) as Promise<Session>;
  }

  // ===== RetryConfig API =====

  async getRetryConfigs(): Promise<RetryConfig[]> {
    return DesktopApp.GetRetryConfigs() as Promise<RetryConfig[]>;
  }

  async getRetryConfig(id: number): Promise<RetryConfig> {
    return DesktopApp.GetRetryConfig(id) as Promise<RetryConfig>;
  }

  async createRetryConfig(data: CreateRetryConfigData): Promise<RetryConfig> {
    await DesktopApp.CreateRetryConfig(data as any);
    const configs = await this.getRetryConfigs();
    return configs[configs.length - 1];
  }

  async updateRetryConfig(id: number, data: Partial<RetryConfig>): Promise<RetryConfig> {
    await DesktopApp.UpdateRetryConfig({ ...data, id } as any);
    return this.getRetryConfig(id);
  }

  async deleteRetryConfig(id: number): Promise<void> {
    await DesktopApp.DeleteRetryConfig(id);
  }

  // ===== RoutingStrategy API =====

  async getRoutingStrategies(): Promise<RoutingStrategy[]> {
    return DesktopApp.GetRoutingStrategies() as Promise<RoutingStrategy[]>;
  }

  async getRoutingStrategy(id: number): Promise<RoutingStrategy> {
    return DesktopApp.GetRoutingStrategy(id) as Promise<RoutingStrategy>;
  }

  async createRoutingStrategy(data: CreateRoutingStrategyData): Promise<RoutingStrategy> {
    await DesktopApp.CreateRoutingStrategy(data as any);
    const strategies = await this.getRoutingStrategies();
    return strategies[strategies.length - 1];
  }

  async updateRoutingStrategy(id: number, data: Partial<RoutingStrategy>): Promise<RoutingStrategy> {
    await DesktopApp.UpdateRoutingStrategy({ ...data, id } as any);
    return this.getRoutingStrategy(id);
  }

  async deleteRoutingStrategy(id: number): Promise<void> {
    await DesktopApp.DeleteRoutingStrategy(id);
  }

  // ===== ProxyRequest API =====

  async getProxyRequests(params?: CursorPaginationParams): Promise<CursorPaginationResult<ProxyRequest>> {
    const result = await DesktopApp.GetProxyRequestsCursor(
      params?.limit ?? 100,
      params?.before ?? 0,
      params?.after ?? 0
    );
    return result as CursorPaginationResult<ProxyRequest>;
  }

  async getProxyRequestsCount(): Promise<number> {
    return DesktopApp.GetProxyRequestsCount();
  }

  async getProxyRequest(id: number): Promise<ProxyRequest> {
    return DesktopApp.GetProxyRequest(id) as Promise<ProxyRequest>;
  }

  async getProxyUpstreamAttempts(proxyRequestId: number): Promise<ProxyUpstreamAttempt[]> {
    return DesktopApp.GetProxyUpstreamAttempts(proxyRequestId) as Promise<ProxyUpstreamAttempt[]>;
  }

  // ===== Proxy Status API =====

  async getProxyStatus(): Promise<ProxyStatus> {
    return DesktopApp.GetProxyStatus() as Promise<ProxyStatus>;
  }

  // ===== Provider Stats API =====

  async getProviderStats(clientType?: string, projectId?: number): Promise<Record<number, ProviderStats>> {
    return DesktopApp.GetProviderStats(clientType ?? '', projectId ?? 0) as Promise<Record<number, ProviderStats>>;
  }

  // ===== Settings API =====

  async getSettings(): Promise<Record<string, string>> {
    return DesktopApp.GetSettings();
  }

  async getSetting(key: string): Promise<{ key: string; value: string }> {
    const value = await DesktopApp.GetSetting(key);
    return { key, value };
  }

  async updateSetting(key: string, value: string): Promise<{ key: string; value: string }> {
    await DesktopApp.UpdateSetting(key, value);
    return { key, value };
  }

  async deleteSetting(key: string): Promise<void> {
    await DesktopApp.DeleteSetting(key);
  }

  // ===== Logs API =====

  async getLogs(limit = 100): Promise<{ lines: string[]; count: number }> {
    return DesktopApp.GetLogs(limit) as Promise<{ lines: string[]; count: number }>;
  }

  // ===== Antigravity API =====

  async validateAntigravityToken(refreshToken: string): Promise<AntigravityTokenValidationResult> {
    return DesktopApp.ValidateAntigravityToken(refreshToken) as Promise<AntigravityTokenValidationResult>;
  }

  async validateAntigravityTokens(tokens: string[]): Promise<AntigravityBatchValidationResult> {
    return DesktopApp.ValidateAntigravityTokens(tokens) as Promise<AntigravityBatchValidationResult>;
  }

  async validateAntigravityTokenText(tokenText: string): Promise<AntigravityBatchValidationResult> {
    return DesktopApp.ValidateAntigravityTokenText(tokenText) as Promise<AntigravityBatchValidationResult>;
  }

  async getAntigravityProviderQuota(providerId: number, forceRefresh?: boolean): Promise<AntigravityQuotaData> {
    return DesktopApp.GetAntigravityProviderQuota(providerId, forceRefresh ?? false) as Promise<AntigravityQuotaData>;
  }

  async startAntigravityOAuth(): Promise<{ authURL: string; state: string }> {
    return DesktopApp.StartAntigravityOAuth() as Promise<{ authURL: string; state: string }>;
  }

  // ===== Cooldown API =====

  async getCooldowns(): Promise<Cooldown[]> {
    return DesktopApp.GetCooldowns() as Promise<Cooldown[]>;
  }

  async clearCooldown(providerId: number): Promise<void> {
    await DesktopApp.ClearCooldown(providerId);
  }

  // ===== Wails Events 订阅 =====

  subscribe<T = unknown>(eventType: WSMessageType, callback: EventCallback<T>): UnsubscribeFn {
    // 保存回调
    if (!this.eventCallbacks.has(eventType)) {
      this.eventCallbacks.set(eventType, new Set());
    }
    this.eventCallbacks.get(eventType)!.add(callback as EventCallback);

    // 如果这是该事件类型的第一个订阅者，设置 Wails 事件监听
    if (!this.eventUnsubscribers.has(eventType)) {
      this.setupWailsEventListener(eventType);
    }

    return () => {
      this.eventCallbacks.get(eventType)?.delete(callback as EventCallback);

      // 如果没有更多订阅者，取消 Wails 事件监听
      if (this.eventCallbacks.get(eventType)?.size === 0) {
        this.eventUnsubscribers.get(eventType)?.();
        this.eventUnsubscribers.delete(eventType);
      }
    };
  }

  private setupWailsEventListener(eventType: WSMessageType): void {
    if (!window.runtime?.EventsOn) {
      console.warn('[WailsTransport] runtime.EventsOn not available');
      return;
    }

    const unsubscribe = window.runtime.EventsOn(eventType, (data: unknown) => {
      const callbacks = this.eventCallbacks.get(eventType);
      callbacks?.forEach((callback) => callback(data));
    });

    this.eventUnsubscribers.set(eventType, unsubscribe);
  }

  // ===== 生命周期 =====

  async connect(): Promise<void> {
    this.connected = true;
  }

  disconnect(): void {
    // 清理所有事件监听
    this.eventUnsubscribers.forEach((unsubscribe) => unsubscribe());
    this.eventUnsubscribers.clear();
    this.eventCallbacks.clear();
    this.connected = false;
  }

  isConnected(): boolean {
    return this.connected;
  }
}
