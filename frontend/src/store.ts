// A minimal hand-rolled reactive store: a value plus a set of subscribers
// notified on change. No framework, no dependency — the game screen's UI
// surface (map, buttons, data) is bounded enough that a compiler-driven
// reactivity system buys less than it costs here.
export type Unsubscribe = () => void;

export function createStore<T>(initial: T) {
	let value = initial;
	const subscribers = new Set<(value: T) => void>();

	return {
		get: (): T => value,
		set: (next: T): void => {
			value = next;
			for (const subscriber of subscribers) subscriber(value);
		},
		// subscribe calls fn immediately with the current value, then again
		// on every future change — callers don't need a separate initial
		// render plus a change handler, just one function.
		subscribe: (fn: (value: T) => void): Unsubscribe => {
			subscribers.add(fn);
			fn(value);
			return () => subscribers.delete(fn);
		},
	};
}
