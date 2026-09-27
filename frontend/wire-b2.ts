const fs = require('fs');

function append(path, text) {
  let s = fs.readFileSync(path, 'utf8');
  const eol = s.includes('\r\n') ? '\r\n' : '\n';
  if (!s.endsWith(eol)) s += eol;
  s += text.replace(/\n/g, eol) + eol;
  fs.writeFileSync(path, s);
  console.log('appended to ' + path);
}

function patch(path, anchors) {
  let s = fs.readFileSync(path, 'utf8');
  const eol = s.includes('\r\n') ? '\r\n' : '\n';
  for (const [from, to] of anchors) {
    const f = from.replace(/\n/g, eol);
    const t = to.replace(/\n/g, eol);
    if (!s.includes(f)) { console.error('anchor not found in ' + path + ': ' + JSON.stringify(from)); process.exit(1); }
    s = s.replace(f, t);
  }
  fs.writeFileSync(path, s);
  console.log('patched ' + path);
}

// API client: train any player (Tier B2).
append('src/services/api/prodigies.ts', `
// Tier B (B2): train any squad player, not just the twelve prodigies. The
// backend registers a growth profile on demand for untracked players.
export function trainPlayer(playerId: string, focus: string): Promise<TrainProdigyResponse> {
  return apiFetch<TrainProdigyResponse>(
    \`/players/\${encodeURIComponent(playerId)}/train\`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ focus }),
    },
    { status: 'error', message: 'Network error running the training session.' },
  );
}
`);

// PlayerSheetModal: training control section (Tier B2).
patch('src/components/clubs/PlayerSheet/PlayerSheetModal.tsx', [
  [
    "import { ovrTone, positionTone } from '../../../lib/constants';",
    "import { ovrTone, positionTone, TRAINING_FOCUSES } from '../../../lib/constants';",
  ],
  [
    "import { soundManager } from '../../../audio/webAudio';",
    "import { soundManager } from '../../../audio/webAudio';\nimport { trainPlayer } from '../../../services/api';",
  ],
  [
    "                {/* Medical record (F11): per-player injury history */}",
    "                {/* Training control (Tier B2): one regimen per call from\n                    the shared weekly energy, for any player. */}\n                <div className=\"p-4 rounded-xl border border-line bg-ink/40 space-y-2 text-[13px]\">\n                  <h4 className=\"font-display text-[14px] font-semibold text-bone mb-1\">Training session</h4>\n                  <p className=\"text-[11px] text-sage\">Run one focused session. Every session spends from the shared weekly training energy.</p>\n                  <div className=\"flex flex-wrap gap-2\">\n                    {TRAINING_FOCUSES.map((focus) => (\n                      <button\n                        key={focus.value}\n                        type=\"button\"\n                        onClick={() => {\n                          soundManager.playClick();\n                          void trainPlayer(player.player_id, focus.value).then((res) => {\n                            setTrainingResult(res.status === 'error' ? (res.message ?? 'Training failed.') : `Session complete · OVR ${res.ovr ?? player.ovr}`);\n                          });\n                        }}\n                        className=\"min-h-9 border border-line bg-cardLight px-3 text-[12px] font-semibold text-sage hover:border-brass/30 hover:text-bone\"\n                      >\n                        {focus.label}\n                      </button>\n                    ))}\n                  </div>\n                  {trainingResult && <p className=\"text-[12px] text-brass\">{trainingResult}</p>}\n                </div>\n\n                {/* Medical record (F11): per-player injury history */}",
  ],
]);
