/**
 * 主题配置
 * 统一的颜色管理系统 - 所有颜色从 CSS 变量获取
 */

/**
 * Provider 类型定义
 */
export type ProviderType =
  | 'anthropic'
  | 'openai'
  | 'deepseek'
  | 'google'
  | 'azure'
  | 'aws'
  | 'cohere'
  | 'mistral'
  | 'custom'
  | 'antigravity'
  | 'kiro';

/**
 * Client 类型定义
 */
export type ClientType = 'claude' | 'openai' | 'codex' | 'gemini';

/**
 * Theme mode types
 */
export type ThemeMode = 'light' | 'dark' | 'system';

/**
 * Luxury theme types
 */
export type LuxuryTheme =
  | 'hermes'
  | 'tiffany'
  | 'chanel'
  | 'cartier'
  | 'burberry'
  | 'gucci'
  | 'dior';

/**
 * All available themes
 */
export type Theme = ThemeMode | LuxuryTheme;

/**
 * Theme metadata interface
 */
export interface ThemeMetadata {
  id: Theme;
  name: string;
  description: string;
  baseMode: 'light' | 'dark';
  category: 'default' | 'luxury';
  brandInspiration?: string;
  accentColor: string;
  primaryColor: string;
  secondaryColor: string;
}

/**
 * Theme registry with metadata for all themes
 */
export const THEME_REGISTRY: Record<Theme, ThemeMetadata> = {
  light: {
    id: 'light',
    name: 'Light',
    description: 'Clean and bright',
    baseMode: 'light',
    category: 'default',
    accentColor: 'oklch(0.9772 0 0)', // Light gray background
    primaryColor: 'oklch(0.3261 0 0)', // Black
    secondaryColor: 'oklch(0.9772 0 0)', // Light gray
  },
  dark: {
    id: 'dark',
    name: 'Dark',
    description: 'Easy on the eyes',
    baseMode: 'dark',
    category: 'default',
    accentColor: 'oklch(0.2741 0.0055 286.0329)', // Dark gray background
    primaryColor: 'oklch(0.9848 0 0)', // White
    secondaryColor: 'oklch(0.2741 0.0055 286.0329)', // Dark gray
  },
  system: {
    id: 'system',
    name: 'System',
    description: 'Matches your system preference',
    baseMode: 'light',
    category: 'default',
    accentColor: 'oklch(0.5 0 0)', // Medium gray (represents both light and dark)
    primaryColor: 'oklch(0.3261 0 0)', // Black
    secondaryColor: 'oklch(0.9772 0 0)', // Light gray
  },
  hermes: {
    id: 'hermes',
    name: 'Hermès',
    description: 'Warm sophistication with iconic orange',
    baseMode: 'light',
    category: 'luxury',
    brandInspiration: 'Hermès',
    accentColor: 'oklch(0.65 0.15 55)',
    primaryColor: 'oklch(0.65 0.15 55)',
    secondaryColor: 'oklch(0.45 0.08 50)',
  },
  tiffany: {
    id: 'tiffany',
    name: 'Tiffany',
    description: 'Elegant robin\'s egg blue',
    baseMode: 'light',
    category: 'luxury',
    brandInspiration: 'Tiffany & Co.',
    accentColor: 'oklch(0.70 0.10 195)',
    primaryColor: 'oklch(0.70 0.10 195)',
    secondaryColor: 'oklch(0.75 0.01 240)',
  },
  chanel: {
    id: 'chanel',
    name: 'Chanel',
    description: 'Timeless black and white elegance',
    baseMode: 'dark',
    category: 'luxury',
    brandInspiration: 'Chanel',
    accentColor: 'oklch(0.75 0.12 85)', // Gold accent
    primaryColor: 'oklch(0.98 0.005 280)', // White
    secondaryColor: 'oklch(0.25 0.01 280)', // Black
  },
  cartier: {
    id: 'cartier',
    name: 'Cartier',
    description: 'Rich burgundy with gold accents',
    baseMode: 'dark',
    category: 'luxury',
    brandInspiration: 'Cartier',
    accentColor: 'oklch(0.75 0.14 85)', // Gold accent
    primaryColor: 'oklch(0.45 0.18 20)', // Burgundy red
    secondaryColor: 'oklch(0.70 0.12 80)', // Gold
  },
  burberry: {
    id: 'burberry',
    name: 'Burberry',
    description: 'Classic heritage tan',
    baseMode: 'light',
    category: 'luxury',
    brandInspiration: 'Burberry',
    accentColor: 'oklch(0.50 0.18 25)', // Red accent
    primaryColor: 'oklch(0.60 0.08 65)', // Tan/Beige
    secondaryColor: 'oklch(0.25 0.01 280)', // Black
  },
  gucci: {
    id: 'gucci',
    name: 'Gucci',
    description: 'Bold forest green with gold',
    baseMode: 'dark',
    category: 'luxury',
    brandInspiration: 'Gucci',
    accentColor: 'oklch(0.72 0.13 82)', // Gold accent
    primaryColor: 'oklch(0.40 0.12 155)', // Forest green
    secondaryColor: 'oklch(0.45 0.18 20)', // Red
  },
  dior: {
    id: 'dior',
    name: 'Dior',
    description: 'Soft understated elegance',
    baseMode: 'light',
    category: 'luxury',
    brandInspiration: 'Dior',
    accentColor: 'oklch(0.68 0.08 25)', // Rose gold accent
    primaryColor: 'oklch(0.55 0.02 260)', // Gray
    secondaryColor: 'oklch(0.70 0.01 250)', // Light gray
  },
};

/**
 * Get theme metadata
 */
export function getThemeMetadata(theme: Theme): ThemeMetadata {
  return THEME_REGISTRY[theme];
}

/**
 * Check if theme is a luxury theme
 */
export function isLuxuryTheme(theme: Theme): boolean {
  return THEME_REGISTRY[theme].category === 'luxury';
}

/**
 * Get the base mode (light/dark) for a theme
 */
export function getThemeBaseMode(theme: Theme): 'light' | 'dark' {
  return THEME_REGISTRY[theme].baseMode;
}

/**
 * Get all luxury themes
 */
export function getLuxuryThemes(): ThemeMetadata[] {
  return Object.values(THEME_REGISTRY).filter(t => t.category === 'luxury');
}

/**
 * Get all default themes
 */
export function getDefaultThemes(): ThemeMetadata[] {
  return Object.values(THEME_REGISTRY).filter(t => t.category === 'default');
}

/**
 * 颜色变量名称类型（所有可用的 CSS 变量）
 */
export type ColorVariable =
  | 'background'
  | 'foreground'
  | 'primary'
  | 'secondary'
  | 'border'
  | 'success'
  | 'warning'
  | 'error'
  | 'info'
  | `provider-${ProviderType}`
  | `client-${ClientType}`;

// 保留旧的 colors 对象用于向后兼容（已弃用，将在未来版本移除）
/** @deprecated 使用 CSS 变量和工具函数替代 */
export const colors = {
  background: '#1E1E1E',
  surfacePrimary: '#252526',
  surfaceSecondary: '#2D2D30',
  surfaceHover: '#3C3C3C',
  border: '#3C3C3C',
  textPrimary: '#CCCCCC',
  textSecondary: '#8C8C8C',
  textMuted: '#5A5A5A',
  accent: '#0078D4',
  accentHover: '#1084D9',
  accentSubtle: 'rgba(0, 120, 212, 0.15)',
  success: '#4EC9B0',
  warning: '#DDB359',
  error: '#F14C4C',
  info: '#4FC1FF',
  providers: {
    anthropic: '#D4A574',
    openai: '#10A37F',
    deepseek: '#4A90D9',
    google: '#4285F4',
    azure: '#0089D6',
    aws: '#FF9900',
    cohere: '#D97706',
    mistral: '#F97316',
    custom: '#8C8C8C',
  },
} as const;

// 间距系统
export const spacing = {
  xs: '4px',
  sm: '8px',
  md: '12px',
  lg: '16px',
  xl: '24px',
  xxl: '32px',
} as const;

// 排版系统
export const typography = {
  caption: { size: '11px', lineHeight: '1.4', weight: 400 },
  body: { size: '13px', lineHeight: '1.5', weight: 400 },
  headline: { size: '15px', lineHeight: '1.4', weight: 600 },
  title3: { size: '17px', lineHeight: '1.3', weight: 600 },
  title2: { size: '20px', lineHeight: '1.2', weight: 700 },
  title1: { size: '24px', lineHeight: '1.2', weight: 700 },
  largeTitle: { size: '28px', lineHeight: '1.1', weight: 700 },
} as const;

// 圆角
export const borderRadius = {
  sm: '4px',
  md: '8px',
  lg: '12px',
} as const;

// 阴影
export const shadows = {
  card: '0 2px 8px rgba(0, 0, 0, 0.3)',
  cardHover: '0 4px 12px rgba(0, 0, 0, 0.4)',
} as const;

/**
 * 从 CSS 变量获取计算后的颜色值
 *
 * @param varName - CSS 变量名称（不含 -- 前缀）
 * @param element - 可选的 DOM 元素，默认为 document.documentElement
 * @returns 计算后的颜色值（如 "oklch(0.7324 0.0867 56.4182)"）
 *
 * @example
 * const anthropicColor = getComputedColor('provider-anthropic')
 * // 返回: "oklch(0.7324 0.0867 56.4182)"
 */
export function getComputedColor(
  varName: ColorVariable,
  element: HTMLElement = document.documentElement,
): string {
  return getComputedStyle(element).getPropertyValue(`--${varName}`).trim();
}

/**
 * 获取 Provider 的品牌色 CSS 变量名
 *
 * @param provider - Provider 类型
 * @returns CSS 变量引用字符串（如 "var(--provider-anthropic)"）
 *
 * @example
 * const colorVar = getProviderColorVar('anthropic')
 * // 返回: "var(--provider-anthropic)"
 *
 * // 用于组件样式
 * <div style={{ color: getProviderColorVar(provider.type) }}>
 */
export function getProviderColorVar(provider: ProviderType): string {
  return `var(--provider-${provider})`;
}

/**
 * 获取 Provider 的计算后颜色值
 *
 * @param provider - Provider 类型
 * @returns 计算后的颜色值
 *
 * @example
 * const color = getProviderColor('anthropic')
 * // 用于需要实际颜色值的场景（如 SVG fill、第三方库）
 */
export function getProviderColor(provider: ProviderType): string {
  return getComputedColor(`provider-${provider}`);
}

/**
 * 获取 Provider 显示名称
 */
export function getProviderDisplayName(type: string): string {
  const names: Record<string, string> = {
    anthropic: 'Anthropic',
    openai: 'OpenAI',
    deepseek: 'DeepSeek',
    google: 'Google',
    azure: 'Azure',
    aws: 'AWS Bedrock',
    cohere: 'Cohere',
    mistral: 'Mistral',
    custom: 'Custom',
  };
  return names[type.toLowerCase()] || type;
}

/**
 * 获取 Client 的品牌色 CSS 变量名
 *
 * @param client - Client 类型
 * @returns CSS 变量引用字符串
 *
 * @example
 * const colorVar = getClientColorVar('claude')
 * // 返回: "var(--client-claude)"
 */
export function getClientColorVar(client: ClientType): string {
  return `var(--client-${client})`;
}

/**
 * 获取 Client 的计算后颜色值
 *
 * @param client - Client 类型
 * @returns 计算后的颜色值
 *
 * @example
 * const color = getClientColor('claude')
 */
export function getClientColor(client: ClientType): string {
  return getComputedColor(`client-${client}`);
}

/**
 * 为颜色添加透明度（用于背景等场景）
 *
 * @param color - OKLCh 格式的颜色字符串
 * @param opacity - 透明度（0-1）
 * @returns 带透明度的颜色字符串
 *
 * @example
 * const bgColor = withOpacity(getProviderColor('anthropic'), 0.2)
 * // 返回: "oklch(0.7324 0.0867 56.4182 / 0.2)"
 */
export function withOpacity(color: string, opacity: number): string {
  // 处理 oklch(...) 格式
  if (color.startsWith('oklch(')) {
    const inner = color.slice(6, -1); // 移除 "oklch(" 和 ")"
    return `oklch(${inner} / ${opacity})`;
  }

  // 处理其他格式（HEX、RGB 等）- 降级处理
  console.warn(`withOpacity: 不支持的颜色格式 "${color}"，建议使用 OKLCh 格式`);
  return color;
}

/** @deprecated 使用 getClientColorVar 或 getClientColor 替代 */
export const clientColors: Record<string, string> = {
  claude: colors.providers.anthropic,
  openai: colors.providers.openai,
  codex: colors.providers.openai,
  gemini: colors.providers.google,
};
