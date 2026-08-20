(function () {
  try {
    var k = "task-tracker-theme";
    var s = null;
    try {
      s = window.localStorage.getItem(k);
    } catch (e) {}
    var t = s === "light" || s === "dark" ? s : "system";
    var d =
      t === "dark" ||
      (t === "system" &&
        window.matchMedia &&
        window.matchMedia("(prefers-color-scheme: dark)").matches);
    document.documentElement.classList.toggle("dark", d);
    document.documentElement.style.colorScheme = d ? "dark" : "light";
  } catch (e) {}
})();
