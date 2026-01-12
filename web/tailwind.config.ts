import type { Config } from 'tailwindcss';

export default {
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx}',
  ],
  darkMode: 'class',
  theme: {
    extend: {
      spacing: {
        'xs': '4px',
        'sm': '8px',
        'md': '12px',
        'lg': '16px',
        'xl': '24px',
        'xxl': '32px',
      },
      borderRadius: {
        'sm': '4px',
        'md': '8px',
        'lg': '12px',
      },
      fontSize: {
        'caption': ['11px', { lineHeight: '1.4' }],
        'body': ['13px', { lineHeight: '1.5' }],
        'headline': ['15px', { lineHeight: '1.4' }],
        'title3': ['17px', { lineHeight: '1.3' }],
        'title2': ['20px', { lineHeight: '1.2' }],
        'title1': ['24px', { lineHeight: '1.2' }],
        'large-title': ['28px', { lineHeight: '1.1' }],
      },
      colors: {
        background: '#f8f4f0',
        'surface-primary': '#ffffff',
        'surface-secondary': '#ebe7e3',
        'surface-hover': '#e0dbd6',
        border: '#d4cfc9',
        'text-primary': '#3d3d3d',
        'text-secondary': '#6b6b6b',
        'text-muted': '#9a9a9a',
        accent: {
          DEFAULT: '#0078D4',
          hover: '#1084D9',
        },
        success: '#4EC9B0',
        warning: '#DDB359',
        error: '#F14C4C',
        info: '#4FC1FF',
      },
      boxShadow: {
        'card': '0 2px 8px rgba(0, 0, 0, 0.08)',
        'card-hover': '0 4px 12px rgba(0, 0, 0, 0.12)',
      },
      animation: {
        'snowfall': 'snowfall 8s linear infinite',
        'spin-slow': 'spin 3s linear infinite',
      },
      keyframes: {
        snowfall: {
          '0%': { transform: 'translateY(-10px) translateX(-10px) rotate(0deg)', opacity: '0' },
          '20%': { opacity: '1' },
          '100%': { transform: 'translateY(8rem) translateX(10px) rotate(180deg)', opacity: '0' },
        },
      },
    },
  },
} satisfies Config;
