import { useEffect } from "react";

export function useDebouncedEffect(
    effect: () => void,
    deps: React.DependencyList,
    delay: number,
) {
    useEffect(() => {
        const handle = window.setTimeout(effect, delay);
        return () => window.clearTimeout(handle);
    }, deps); // eslint-disable-line react-hooks/exhaustive-deps
}
