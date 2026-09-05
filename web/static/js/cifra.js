(function () {
  function boot() {
    const hook = window.amarra && window.amarra.hook;
    if (!hook || typeof hook.register !== "function") return;

    hook.register("theme", {
      connect(el) {
        const KEY = "cifra-theme";
        const apply = (light) => {
          document.documentElement.classList.toggle("light", light);
          try {
            localStorage.setItem(KEY, light ? "light" : "dark");
          } catch (_) {}
          const meta = document.querySelector('meta[name="theme-color"]');
          if (meta) meta.setAttribute("content", light ? "#f7f1e2" : "#121816");
          el.setAttribute("aria-pressed", light ? "true" : "false");
        };
        el.addEventListener("click", () => {
          apply(!document.documentElement.classList.contains("light"));
        });
      },
    });

    hook.register("password", {
      connect(el) {
        el.addEventListener("click", () => {
          const id = el.getAttribute("data-amarra-password-for");
          const input = id ? document.querySelector(id) : el.parentElement?.querySelector("input");
          if (!input) return;
          const show = input.type === "password";
          input.type = show ? "text" : "password";
          el.setAttribute("aria-pressed", show ? "true" : "false");
        });
      },
    });

    hook.register("reveal", {
      connect(el) {
        const sync = () => {
          const show = el.getAttribute("data-amarra-reveal-show") || "";
          const target = document.querySelector(el.getAttribute("data-amarra-reveal-target") || "");
          if (!target) return;
          target.hidden = el.value !== show;
        };
        el.addEventListener("change", sync);
        sync();
      },
    });

    hook.scan(document);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
