// Sign-in page (/login): theme toggle, "remember me" checkbox styling and the
// fetch-based sign-in with the two-step verification challenge. Copy comes
// from the form's data-msg-* attributes, rendered in the page locale.
(() => {
  const form = document.getElementById("login-form");
  if (!form) return;
  const status = document.getElementById("login-status");
  const themeButton = document.getElementById("theme-toggle-btn");
  const themeLabel = document.getElementById("theme-toggle-label");
  const remember = document.getElementById("remember-me");
  const formHeader = document.querySelector(".form-header");
  const msg = form.dataset;

  if (remember) {
    remember.addEventListener("change", () => {
      const box = remember.nextElementSibling;
      const svg = box.querySelector("svg");
      box.style.background = remember.checked ? "var(--brand-primary)" : "var(--bg-input)";
      box.style.borderColor = remember.checked ? "var(--brand-primary)" : "var(--border-light)";
      if (svg) svg.style.display = remember.checked ? "block" : "none";
    });
  }

  const updateThemeLabel = () => {
    if (!themeButton || !themeLabel) return;
    const dark = document.documentElement.getAttribute("data-theme") === "dark";
    themeLabel.textContent = dark ? themeButton.dataset.labelDark : themeButton.dataset.labelLight;
  };
  updateThemeLabel();
  themeButton?.addEventListener("click", () => {
    const next = document.documentElement.getAttribute("data-theme") === "dark" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", next);
    try {
      localStorage.setItem("hydro-theme", next);
      localStorage.setItem("hydro-ui-theme-v2", "workbench-" + next);
    } catch (_) {}
    updateThemeLabel();
  });

  const showMessage = (type, text) => {
    status.className = "form-alert alert-" + type;
    status.setAttribute("role", type === "error" ? "alert" : "status");
    status.textContent = text;
    status.style.display = "block";
  };

  const showChallenge = (result) => {
    const challenge = document.getElementById("mfa-challenge");
    const code = document.getElementById("login-mfa-code");
    const countdown = document.querySelector("[data-mfa-countdown]");
    const nextCode = document.querySelector("[data-mfa-next]");
    challenge.hidden = false;
    if (formHeader) formHeader.hidden = true;
    form.querySelectorAll(".input-group, .form-row-utility").forEach((group) => {
      group.hidden = true;
    });
    code.disabled = false;
    code.required = true;
    const period = Math.max(1, Number(result.totp_period_seconds) || 30);
    const serverOffset = Number(result.server_time_unix) * 1000 - Date.now();
    const initialSlot = Math.floor(Number(result.server_time_unix) / period);
    const tick = () => {
      const serverNow = Date.now() + serverOffset;
      const slot = Math.floor(serverNow / 1000 / period);
      const remaining = period - (Math.floor(serverNow / 1000) % period);
      countdown.textContent = msg.msgCodeExpires + String(Math.floor(remaining / 60)).padStart(2, "0") + ":" + String(remaining % 60).padStart(2, "0") + msg.msgCodeExpiresSuffix;
      nextCode.disabled = slot <= initialSlot;
    };
    tick();
    if (!challenge.dataset.timerStarted) {
      challenge.dataset.timerStarted = "true";
      window.setInterval(tick, 250);
      nextCode.addEventListener("click", () => {
        code.value = "";
        code.focus();
        status.textContent = msg.msgNextCode;
      });
    }
    showMessage("warning", msg.msgMfaRequired);
    code.focus();
  };

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const button = form.querySelector('button[type="submit"]');
    const original = [...button.childNodes].map((node) => node.cloneNode(true));
    const restoreButton = () => {
      button.disabled = false;
      button.replaceChildren(...original);
    };
    const busy = document.createElement("span");
    busy.textContent = msg.msgSigningIn;
    button.disabled = true;
    button.replaceChildren(busy);
    status.style.display = "none";

    try {
      const response = await fetch("/auth/login", {
        method: "POST",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          "X-CSRF-Token": form.elements._csrf.value,
          Accept: "application/json",
        },
        body: new URLSearchParams(new FormData(form)),
      });
      let result = {};
      try {
        result = await response.json();
      } catch (_) {}

      if (!response.ok) {
        if (response.status === 401 && result.error === "mfa_code_required") showChallenge(result);
        else if (response.status === 401 && !document.getElementById("mfa-challenge").hidden) showMessage("error", msg.msgMfaInvalid);
        else if (response.status === 401) showMessage("error", msg.msgInvalid);
        else if (response.status === 429) showMessage("error", msg.msgRateLimited);
        else if (response.status === 403) showMessage("error", msg.msgCsrf);
        else showMessage("error", msg.msgUnavailable);
        restoreButton();
        return;
      }

      const notice = result.security_notice || {};
      const attempts = Number(notice.failed_attempts || 0);
      let message = msg.msgSuccess;
      if (notice.previous_login_at) {
        const when = new Date(notice.previous_login_at).toLocaleString(document.documentElement.lang === "tr" ? "tr-TR" : "en-US");
        message = msg.msgPreviousLogin + when + ". " + attempts + msg.msgFailedAttempts;
      } else if (attempts > 0) {
        message = msg.msgFirstLogin + attempts + msg.msgFirstLoginSuffix;
      }
      showMessage(attempts > 0 ? "warning" : "success", message);
      setTimeout(
        () => {
          const target = form.dataset.returnTo || "";
          // Only same-site paths: the server already sanitised return_to.
          window.location.href = target.startsWith("/") && !target.startsWith("//") ? target : "/admin";
        },
        notice.previous_login_at || attempts > 0 ? 1800 : 250,
      );
    } catch (_) {
      showMessage("error", msg.msgUnreachable);
      restoreButton();
    }
  });
})();
