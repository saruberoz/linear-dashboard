// lint-staged runs these from this folder for staged files under frontend/.
export default {
  "*.{ts,tsx,js,mjs}": ["eslint --fix", "prettier --write"],
  "*.{json,css,md}": ["prettier --write"],
  // Type errors can come from files that weren't staged, so check the whole project once.
  "*.{ts,tsx}": () => "tsc --noEmit",
}
