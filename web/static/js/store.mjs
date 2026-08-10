// A minimal reactive store: a value plus a set of subscribers notified on
// change. No framework, no dependency — per docs/game-ui.md's "Frontend
// approach" decision, this is the fallback for state that needs to update
// more than one place at once, reached for only once hand-wiring DOM
// updates directly stops being manageable.
export function createStore(initial) {
	let value = initial;
	const subscribers = new Set();

	return {
		get: () => value,
		set: (next) => {
			value = next;
			for (const subscriber of subscribers) subscriber(value);
		},
		// subscribe calls fn immediately with the current value, then again
		// on every future change — callers don't need a separate initial
		// render plus a change handler, just one function. Returns an
		// unsubscribe function.
		subscribe: (fn) => {
			subscribers.add(fn);
			fn(value);
			return () => subscribers.delete(fn);
		},
	};
}
