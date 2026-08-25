/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./internal/views/**/*.templ",
    "./internal/views/**/*_templ.go",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Poppins', 'ui-sans-serif', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
