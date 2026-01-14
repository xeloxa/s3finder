// Auto-fetch latest version from GitHub releases
(function() {
  const REPO = 'xeloxa/s3finder';
  const FALLBACK_VERSION = 'v1.2.4';

  async function fetchLatestVersion() {
    try {
      const response = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`);
      if (!response.ok) throw new Error('Failed to fetch');
      const data = await response.json();
      return data.tag_name || FALLBACK_VERSION;
    } catch (e) {
      return FALLBACK_VERSION;
    }
  }

  function updateVersionElements(version) {
    // Update sidebar version
    document.querySelectorAll('.sidebar-version').forEach(el => {
      el.textContent = version;
    });

    // Update footer
    document.querySelectorAll('.docs-footer p').forEach(el => {
      el.innerHTML = el.innerHTML.replace(/v\d+\.\d+\.\d+/, version);
    });

    // Update any inline version references
    document.querySelectorAll('[data-version]').forEach(el => {
      el.textContent = version;
    });
  }

  // Run on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', async () => {
      const version = await fetchLatestVersion();
      updateVersionElements(version);
    });
  } else {
    fetchLatestVersion().then(updateVersionElements);
  }
})();
