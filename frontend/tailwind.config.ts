import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#16302A",
        inksoft: "#3A544A",
        paper: "#EEF1E9",
        line: "#D9DECE",
        brass: "#A8813F",
        brassdk: "#8A6A2E",
        pen: "#B0342A",
        good: "#2F6B4F",
      },
      fontFamily: {
        serif: ["Georgia", "Palatino Linotype", "Palatino", "serif"],
        mono: ["ui-monospace", "Menlo", "monospace"],
      },
    },
  },
  plugins: [],
};
export default config;
