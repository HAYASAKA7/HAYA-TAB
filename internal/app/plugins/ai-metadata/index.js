function isEmpty(s) {
  return !s || !s.trim();
}

function safeJSONParse(s) {
  try { return JSON.parse(s); } catch (e) {
    log('[ai-metadata] JSON parse error: ' + e);
    return null;
  }
}

// Per-sync-run counter. Go side (PluginManager.StartSyncRun) resets this to 0
// at the beginning of each TriggerSync invocation. We keep the fallback here
// in case the variable was not initialized for some reason.
if (typeof requestCountThisRun !== 'number') {
  requestCountThisRun = 0;
}

function getConfidenceThreshold() {
  if (!config || typeof config.confidenceThreshold !== 'string') {
    return 0.5;
  }
  var t = parseFloat(config.confidenceThreshold);
  if (isNaN(t) || t < 0 || t > 1) {
    log('[ai-metadata] invalid confidenceThreshold in config, falling back to 0.5');
    return 0.5;
  }
  return t;
}

function getMaxRequestsPerRun() {
  if (!config || typeof config.maxRequestsPerRun !== 'string') {
    return 50;
  }
  var n = parseInt(config.maxRequestsPerRun, 10);
  if (isNaN(n) || n <= 0) {
    log('[ai-metadata] invalid maxRequestsPerRun in config, falling back to 50');
    return 50;
  }
  return n;
}

function parseBooleanString(v, defaultValue) {
  if (typeof v !== 'string') return defaultValue;
  var s = v.trim().toLowerCase();
  if (s === 'true' || s === '1' || s === 'yes' || s === 'on') return true;
  if (s === 'false' || s === '0' || s === 'no' || s === 'off') return false;
  return defaultValue;
}

function isLocalizedOutputEnabled() {
  // Backward-compatible default: false
  // config.localizedOutputEnabled: "true" | "false"
  if (!config) return false;
  return parseBooleanString(config.localizedOutputEnabled, false);
}

function getAppLanguage(tab) {
  // Use application current language from tab.language.
  if (tab && typeof tab.language === 'string' && tab.language.trim()) {
    return tab.language.trim();
  }
  return 'en';
}

module.exports.enhanceMetadata = function (tab) {
  var maxPerRun = getMaxRequestsPerRun();
  if (requestCountThisRun >= maxPerRun) {
    log('[ai-metadata] skipped: reached maxRequestsPerRun=' + maxPerRun);
    return tab;
  }

  // If the plugin is not configured, do nothing.
  if (!config || !config.apiKey || !config.model || !config.baseUrl) {
    log('[ai-metadata] skipped: missing config (baseUrl/model/apiKey).');
    return tab;
  }

  var fileName = '';
  if (tab.filePath) {
    var parts = tab.filePath.split(/[\\/]/);
    fileName = parts[parts.length - 1];
  }

  log(
    '[ai-metadata] invoked for file="' + fileName +
      '" title="' + (tab.title || '') +
      '" artist="' + (tab.artist || '') + '"'
  );

  // If we already have a reasonably complete title and artist, we can choose
  // to only fill missing album, to reduce unnecessary AI calls.
  var hasTitle = !isEmpty(tab.title);
  var hasArtist = !isEmpty(tab.artist);
  if (hasTitle && hasArtist && !isEmpty(tab.album)) {
    log('[ai-metadata] skipped: title/artist/album already present.');
    return tab;
  }

  var userTitle = tab.title || '';
  var userArtist = tab.artist || '';
  var userAlbum = tab.album || '';
  var localizedOutputEnabled = isLocalizedOutputEnabled();
  var appLanguage = getAppLanguage(tab);

  log(
    '[ai-metadata] language mode: localizedOutputEnabled=' + localizedOutputEnabled +
      ' appLanguage=' + appLanguage
  );

  var prompt =
    'You are a music metadata assistant for a guitar tablature library.\n' +
    'You know many popular songs, artists and albums.\n' +
    'Given a raw file name and rough metadata extracted from the file,\n' +
    'infer the most likely canonical song metadata.\n' +
    '\n' +
    'Field definitions:\n' +
    '- title: canonical song title only. No artist, no album, no extra comments.\n' +
    '- artist: primary artist or band only.\n' +
    '- album: official album title if confident; otherwise keep existing value or empty string.\n' +
    '- originCountry: two-letter ISO country code if known; otherwise empty string.\n' +
    '- tags: 0-6 short tags for genre/instrument/mood only.\n' +
    '- confidence: number between 0 and 1.\n' +
    '\n' +
    'General requirements:\n' +
    '1. Do not invent random songs or artists.\n' +
    '2. If uncertain, keep original values for uncertain fields and reduce confidence.\n' +
    '3. Clean filename noise such as "cover", "live", quality tags, website names, and file extensions.\n' +
    '4. Return STRICT JSON only.\n' +
    '5. Output JSON shape MUST be: {"title":"...","artist":"...","album":"...","originCountry":"..","tags":[],"confidence":0.x}.\n' +
    '6. Never duplicate the same content across title/artist/album/tags.\n';

  if (localizedOutputEnabled) {
    prompt +=
      '\n' +
      'Language requirements (localized mode ON):\n' +
      '- title, artist, album MUST be returned in the song release-country language/script when available.\n' +
      '- tags MUST be returned in the user app language: "' + appLanguage + '".\n' +
      '- If release-country language form is unknown, use the most canonical native/publicly recognized form.\n';
  } else {
    prompt +=
      '\n' +
      'Language requirements (localized mode OFF):\n' +
      '- title, artist, album should use the most globally recognizable canonical form.\n' +
      '- tags should be in English.\n';
  }

  var userMessage =
    'File name: ' + fileName + '\n' +
    'Initial title: ' + userTitle + '\n' +
    'Initial artist: ' + userArtist + '\n' +
    'Initial album: ' + userAlbum + '\n' +
    'App language: ' + appLanguage + '\n' +
    'Localized mode: ' + localizedOutputEnabled + '\n';

  var payload = {
    model: config.model,
    messages: [
      { role: 'system', content: prompt },
      { role: 'user', content: userMessage }
    ],
    temperature: 0.3
  };

  log(
    '[ai-metadata] sending AI request: baseUrl=' + config.baseUrl +
      ' model=' + config.model
  );
  requestCountThisRun++;

  var res = httpRequest({
    method: 'POST',
    url: config.baseUrl + '/chat/completions',
    headers: {
      'Authorization': 'Bearer ' + config.apiKey,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(payload)
  });

  if (!res) {
    log('[ai-metadata] http error: no response object.');
    return tab;
  }

  log('[ai-metadata] http status: ' + res.status);
  if (res.status < 200 || res.status >= 300) {
    log('[ai-metadata] http error response body: ' + (res.body || ''));
    return tab;
  }

  var raw = safeJSONParse(res.body);
  if (!raw || !raw.choices || !raw.choices.length) {
    log('[ai-metadata] response has no choices.');
    return tab;
  }

  var content = raw.choices[0].message && raw.choices[0].message.content;
  if (!content) {
    return tab;
  }

  var info = safeJSONParse(content);
  if (!info) {
    log('[ai-metadata] failed to parse AI content as JSON.');
    return tab;
  }

  var threshold = getConfidenceThreshold();
  log('[ai-metadata] using confidence threshold=' + threshold);

  if (typeof info.confidence === 'number') {
    log('[ai-metadata] AI confidence=' + info.confidence);
    if (info.confidence < threshold) {
      // Too low confidence, keep original metadata.
      log('[ai-metadata] skipped: confidence below threshold.');
      return tab;
    }
  }

  if (!isEmpty(info.title)) {
    tab.title = info.title;
  }
  if (!isEmpty(info.artist)) {
    tab.artist = info.artist;
  }
  if (!isEmpty(info.album)) {
    tab.album = info.album;
  }
  if (!isEmpty(info.originCountry)) {
    tab.originCountry = info.originCountry;
  }
  if (Array.isArray(info.tags) && info.tags.length > 0) {
    // Filter tags to keep them short and avoid dumping long text into a single field.
    var filtered = [];
    var lowerTitle = (tab.title || '').toLowerCase();
    var lowerArtist = (tab.artist || '').toLowerCase();
    for (var i = 0; i < info.tags.length; i++) {
      var t = info.tags[i];
      if (typeof t !== 'string') continue;
      var trimmed = t.trim();
      if (!trimmed) continue;
      // Drop tags that are too long or clearly look like sentences.
      if (trimmed.length > 24) continue;
      // Avoid tags that repeat title or artist text.
      var lower = trimmed.toLowerCase();
      if (lowerTitle && lower.indexOf(lowerTitle) !== -1) continue;
      if (lowerArtist && lower.indexOf(lowerArtist) !== -1) continue;
      filtered.push(trimmed);
      if (filtered.length >= 6) break;
    }
    if (filtered.length > 0) {
      tab.tag = filtered.join(', ');
    }
  }

  log('[ai-metadata] updated metadata for ' + (tab.title || fileName));
  return tab;
};
