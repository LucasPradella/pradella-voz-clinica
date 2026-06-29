import type { Config } from 'tailwindcss'

const config: Config = {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // Navy (ação primária / identidade)
        navy: {
          50: '#e8eef6',
          100: '#c5d4e8',
          200: '#9fb8d8',
          300: '#789bc7',
          400: '#5a85bb',
          500: '#3c6faf',
          600: '#2d5a9a',
          700: '#1e3a5f', // primary brand
          800: '#152d4a',
          900: '#0c1f35',
        },
        // Grafite (texto principal, backgrounds escuros)
        grafite: {
          50: '#f5f5f5',
          100: '#e0e0e0',
          200: '#bdbdbd',
          300: '#9e9e9e',
          400: '#757575',
          500: '#616161',
          600: '#424242', // texto principal
          700: '#303030',
          800: '#212121',
          900: '#121212',
        },
        // Cinza (backgrounds, bordas, estados desabilitados)
        gray: {
          50: '#fafafa',
          100: '#f5f5f5', // background padrão
          200: '#eeeeee',
          300: '#e0e0e0', // bordas
          400: '#bdbdbd',
          500: '#9e9e9e', // texto secundário
          600: '#757575',
          700: '#616161',
          800: '#424242',
          900: '#212121',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
      },
      borderRadius: {
        DEFAULT: '0.5rem',
      },
      // Sem verde: paleta restrita a navy, grafite e cinza
    },
  },
  plugins: [],
}

export default config
