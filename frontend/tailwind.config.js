/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Floodlit night pitch — grass, lamp, live red. Not luxury black/gold.
        ink: '#163226',
        dugout: '#1C382C',
        line: '#2E5344',
        bone: '#F3E6C4',
        sage: '#A7B89A',
        brass: '#F3E6C4',
        ember: '#DC2626',
        pitchtone: '#6F9A78',
        obsidian: '#12261C',
        darkBg: '#12261C',
        cardBg: '#1C382C',
        cardLight: '#244536',
        cardHover: '#2B5140',
        borderMuted: '#2E5344',
        accentBlue: '#3D6B8A',
        accentCyan: '#5A8AAB',
        accentGold: '#F3E6C4',
        accentGreen: '#6F9A78',
        accentRed: '#DC2626',
        accentOrange: '#C47A3A',
        accentPurple: '#6B7280',
        glassBg: 'rgba(28, 56, 44, 0.92)',
        glassLight: 'rgba(36, 69, 54, 0.75)',
        glassHover: 'rgba(43, 81, 64, 0.85)',
        glassBorder: 'rgba(243, 230, 196, 0.16)',
        glassBorderBright: 'rgba(243, 230, 196, 0.45)',
        cyberGold: '#F3E6C4',
        cyanGlow: '#5A8AAB',
        emeraldLight: '#6F9A78',
        coralRed: '#DC2626',
        neonPurple: '#6B7280',
      },
      fontFamily: {
        display: ['"Barlow Condensed"', 'Barlow', 'Arial Narrow', 'sans-serif'],
        sans: ['Barlow', 'Segoe UI', 'sans-serif'],
        mono: ['Barlow', 'Segoe UI', 'sans-serif'],
      },
      boxShadow: {
        raised: '0 16px 40px -24px rgba(8, 20, 14, 0.85)',
        glass: '0 16px 40px -24px rgba(8, 20, 14, 0.85)',
      },
      animation: {
        'pulse-subtle': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'fade-in': 'fadeIn 0.2s ease-out',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
      },
      borderRadius: {
        xl: '4px',
        '2xl': '6px',
      },
    },
  },
  plugins: [],
}
