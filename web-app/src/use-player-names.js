import { useEffect, useRef } from "react";
import { applyPlayerNames } from "./player-names.js";

export function usePlayerNameRefresh(client, ...setters) {
  const targets = useRef(setters); targets.current = setters;
  useEffect(() => {
    const apply = names => { for (const setter of targets.current) setter(value => applyPlayerNames(value, names)); };
    if (client.playerNames.size) apply(client.playerNames);
    return client.onPlayerNames(apply);
  }, [client]);
}
