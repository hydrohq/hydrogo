(function () {
  var AUTH_KEY = 'hydro_member_state';
  var channel = null;
  try {
    if (typeof BroadcastChannel !== 'undefined') {
      channel = new BroadcastChannel('hydro_member_auth');
    }
  } catch (e) {}

  function getStoredMember() {
    try {
      var raw = localStorage.getItem(AUTH_KEY);
      return raw ? JSON.parse(raw) : null;
    } catch (e) {
      return null;
    }
  }

  function setStoredMember(member) {
    try {
      if (member && member.email) {
        localStorage.setItem(AUTH_KEY, JSON.stringify(member));
      } else {
        localStorage.removeItem(AUTH_KEY);
      }
    } catch (e) {}
  }

  function renderHeader(member) {
    var loginLinks = document.querySelectorAll('a[href="/member/login"], a[href^="/member/login?"], .nav-member-badge');
    if (!loginLinks.length) return;

    if (member && member.email) {
      loginLinks.forEach(function (link) {
        if (link.classList.contains('nav-member-badge')) {
          link.textContent = '';
          var icon = document.createElement('span');
          icon.className = 'nav-member-icon';
          icon.textContent = '👤';
          link.appendChild(icon);
          link.appendChild(document.createTextNode(' ' + member.email));
          return;
        }
        var badge = document.createElement('a');
        badge.href = '/member/account';
        badge.className = 'nav-member-badge';
        if (window.location.pathname === '/member/account') {
          badge.classList.add('is-active');
        }
        var icon = document.createElement('span');
        icon.className = 'nav-member-icon';
        icon.textContent = '👤';
        badge.appendChild(icon);
        badge.appendChild(document.createTextNode(' ' + member.email));
        if (link.parentNode) {
          link.parentNode.replaceChild(badge, link);
        }
      });
    } else {
      loginLinks.forEach(function (link) {
        if (link.classList.contains('nav-member-badge')) {
          var a = document.createElement('a');
          a.href = '/member/login';
          a.textContent = 'Giriş';
          if (link.parentNode) {
            link.parentNode.replaceChild(a, link);
          }
        }
      });
    }
  }

  function verifySessionWithServer() {
    fetch('/member/me', {
      headers: { 'Accept': 'application/json' },
      credentials: 'same-origin'
    })
      .then(function (res) {
        if (res.ok) {
          return res.json();
        }
        if (res.status === 401 || res.status === 403) {
          var unauthenticated = new Error('unauthenticated');
          unauthenticated.code = 'UNAUTHENTICATED';
          throw unauthenticated;
        }
        var unavailable = new Error('verification unavailable');
        unavailable.code = 'VERIFICATION_UNAVAILABLE';
        throw unavailable;
      })
      .then(function (data) {
        if (data && data.email) {
          var member = { email: data.email, id: data.id };
          setStoredMember(member);
          renderHeader(member);
        }
      })
      .catch(function (err) {
        // Only an explicit authentication rejection may clear the optimistic state.
        // Network/5xx failures must not log the user out visually.
        if (err && err.code === 'UNAUTHENTICATED') {
          setStoredMember(null);
          renderHeader(null);
        }
      });
  }

  // 1. Optimistic Render immediately from local state (Zero FOUC)
  var cached = getStoredMember();
  if (cached) {
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', function () {
        renderHeader(cached);
      });
    } else {
      renderHeader(cached);
    }
  }

  // 2. Background server verification
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', verifySessionWithServer);
  } else {
    verifySessionWithServer();
  }

  // 3. Cross-Tab Live Synchronization
  if (channel) {
    channel.onmessage = function (event) {
      if (event && event.data) {
        if (event.data.type === 'LOGIN') {
          setStoredMember(event.data.member);
          renderHeader(event.data.member);
        } else if (event.data.type === 'LOGOUT') {
          setStoredMember(null);
          renderHeader(null);
        }
      }
    };
  }

  window.addEventListener('storage', function (e) {
    if (e.key === AUTH_KEY) {
      var updated = getStoredMember();
      renderHeader(updated);
    }
  });

  // 4. Hook into member login/logout forms if present on page
  document.addEventListener('submit', function (e) {
    var form = e.target;
    if (form && form.getAttribute('data-member-form') !== null) {
      // Upon successful login/register in form handler, notify
      form.addEventListener('hydro:authenticated', function (evt) {
        if (evt.detail && evt.detail.member) {
          setStoredMember(evt.detail.member);
          renderHeader(evt.detail.member);
          if (channel) {
            channel.postMessage({ type: 'LOGIN', member: evt.detail.member });
          }
        }
      });
    }
  });

  window.hydroAuth = {
    setMember: function (member) {
      setStoredMember(member);
      renderHeader(member);
      if (channel) {
        channel.postMessage({ type: 'LOGIN', member: member });
      }
    },
    clearMember: function () {
      setStoredMember(null);
      renderHeader(null);
      if (channel) {
        channel.postMessage({ type: 'LOGOUT' });
      }
    }
  };
})();

(function () {
  var toggle = document.querySelector('.nav-toggle');
  var nav = document.getElementById('primary-nav');
  if (!toggle || !nav) return;
  toggle.addEventListener('click', function () {
    var open = toggle.getAttribute('aria-expanded') === 'true';
    toggle.setAttribute('aria-expanded', String(!open));
    nav.classList.toggle('is-open', !open);
  });
})();
