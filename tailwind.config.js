/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: ['selector', '[data-theme="dark"]'],
  content: [
    "./pkg/views/**/*.go",
  ],
  theme: {
    extend: {},
  },
  plugins: [],
}
