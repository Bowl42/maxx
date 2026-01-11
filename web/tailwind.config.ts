import type { Config } from 'tailwindcss';

export default {
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx}',
  ],
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
        background: '#1E1E1E',
        'surface-primary': '#252526',
        'surface-secondary': '#2D2D30',
        'surface-hover': '#3C3C3C',
        border: '#3C3C3C',
        'text-primary': '#CCCCCC',
        'text-secondary': '#8C8C8C',
        'text-muted': '#5A5A5A',
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
        'card': '0 2px 8px rgba(0, 0, 0, 0.3)',
        'card-hover': '0 4px 12px rgba(0, 0, 0, 0.4)',
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
