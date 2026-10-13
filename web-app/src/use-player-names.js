import { useEffect, useState, useMemo } from "react";
import { applyPlayerNames } from "./player-names.js";

// Derive labels at the view boundary. A late response cannot overwrite the
// current name, and identical names retain the original objects.
export function usePlayerNameState(client, initial) {
  const [value, setValue] = useState(initial);
  const [revision, setRevision] = useState(0);
  useEffect(() => client.onPlayerNames(() => setRevision(value => value + 1)), [client]);
  return [useMemo(() => applyPlayerNames(value, client.playerNames), [value, client, revision]), setValue];
}
