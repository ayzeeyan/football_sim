/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      // Floodlit night pitch — one named value per role, no aliases.
      // brass is trophy brass (#C5A568) and matches the --accent-brass focus
      // colour in index.css; bone is floodlight cream for primary text.
      colors: {
        ink: '#163226',        // floodlit stand — base surface
        dugout: '#1C382C',     // raised card surface
        line: '#2E5344',       // chalk line — borders
        bone: '#F3E6C4',       // floodlight cream — primary text / accent
        brass: '#C5A568',      // trophy brass — focus, active, highlights
        sage: '#A7B89A',       // worn grass — secondary text
        ember: '#DC2626',      // live red — danger, live state
        pitchtone: '#6F9A78',  // pitch green — positive / qualification
        cardBg: '#1C382C',
        cardLight: '#244536',
        cardHover: '#2B5140',
      },
      fontFamily: {
        display: ['"Barlow Condensed"', 'Barlow', 'Arial Narrow', 'sans-serif'],
        sans: ['Barlow', 'Segoe UI', 'sans-serif'],
        mono: ['ui-monospace', '"Cascadia Mono"', '"SF Mono"', 'Menlo', 'Consolas', 'monospace'],
      },
      boxShadow: {
        raised: '0 16px 40px -24px rgba(8, 20, 14, 0.85)',
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
