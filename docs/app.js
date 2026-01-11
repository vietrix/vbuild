(() => {
  const body = document.body;
  const toggle = document.getElementById("langToggle");
  const stored = window.localStorage ? localStorage.getItem("vbuild-lang") : null;
  const initial = stored === "vi" ? "vi" : "en";

  const setLang = (lang) => {
    body.classList.remove("lang-en", "lang-vi");
    body.classList.add(`lang-${lang}`);
    document.documentElement.lang = lang;
    if (window.localStorage) {
      localStorage.setItem("vbuild-lang", lang);
    }
    if (toggle) {
      toggle.setAttribute("aria-pressed", lang === "vi" ? "true" : "false");
    }
  };

  setLang(initial);

  if (toggle) {
    toggle.addEventListener("click", () => {
      const next = body.classList.contains("lang-en") ? "vi" : "en";
      setLang(next);
    });
  }

  const prefersReduced = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  const targets = document.querySelectorAll(".reveal");

  if (prefersReduced) {
    targets.forEach((el) => el.classList.add("is-visible"));
    return;
  }

  const observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (!entry.isIntersecting) {
          return;
        }
        entry.target.classList.add("is-visible");
        observer.unobserve(entry.target);
      });
    },
    { threshold: 0.15 }
  );

  targets.forEach((el) => {
    const delay = el.getAttribute("data-delay");
    if (delay) {
      el.style.transitionDelay = `${delay}s`;
    }
    observer.observe(el);
  });
})();
