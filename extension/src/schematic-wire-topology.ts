/** Official observed wire lines are segment records, not authored create polylines.
 * Confirmed on EasyEDA Pro 3.2.186 by crossing-probe-dev5 (2026-09-15).
 * Keep the original encoding and segment index: synthetic subdivisions must never
 * become electrical junctions. Net names and primitive membership do not union Xs.
 */
export type WireSegment = [number, number, number, number];
export interface WireAnchor { x: number; y: number }
export const WIRE_CONTACT_EPS = 1e-6;
export type WireContact = 'disjoint' | 'proper-cross' | 'endpoint-touch' | 'collinear-overlap' | 'unknown';
export type WireLineEncoding = 'flat-segments' | 'nested-segments' | 'nested-polyline';
export interface ObservedWireSegment {
	seg: WireSegment;
	wirePrimitiveId: string;
	net: string;
	segmentIndex: number;
	rawLine: number[] | number[][];
	rawEncoding: WireLineEncoding;
}

export function parseObservedWireLine(raw: unknown): { segments: WireSegment[]; rawLine: number[] | number[][]; rawEncoding: WireLineEncoding } {
	if (!Array.isArray(raw) || raw.length === 0) throw new Error('Missing observed wire line.');
	const finite = (v: unknown): v is number => typeof v === 'number' && Number.isFinite(v);
	let segments: WireSegment[];
	let rawLine: number[] | number[][];
	let rawEncoding: WireLineEncoding;
	if (raw.every(finite)) {
		// A flat odd-vertex list is ambiguous and has not been observed on the
		// supported host. Do not guess polyline shape from coordinate parity.
		if (raw.length % 4 !== 0) throw new Error('Ambiguous observed flat wire: expected independent four-coordinate segments.');
		rawLine = [...raw]; rawEncoding = 'flat-segments'; segments = [];
		for (let i = 0; i < raw.length; i += 4) segments.push(raw.slice(i, i + 4) as WireSegment);
	} else if (raw.every(r => Array.isArray(r) && r.length === 4 && r.every(finite))) {
		rawLine = raw.map(r => [...r]); rawEncoding = 'nested-segments';
		segments = raw.map(r => [...r] as WireSegment);
	} else if (raw.length >= 2 && raw.every(r => Array.isArray(r) && r.length === 2 && r.every(finite))) {
		rawLine = raw.map(r => [...r]); rawEncoding = 'nested-polyline'; segments = [];
		for (let i = 0; i + 1 < raw.length; i++) segments.push([...raw[i], ...raw[i + 1]] as WireSegment);
	} else throw new Error('Invalid or unknown observed wire encoding.');
	return { segments, rawLine, rawEncoding };
}

export function readObservedWireSegments(wires: Array<{ getState_Line: () => unknown; getState_PrimitiveId?: () => string; getState_Net?: () => string }>): ObservedWireSegment[] {
	if (!Array.isArray(wires)) throw new Error('Wire enumeration did not return an array.');
	return wires.flatMap(w => {
		const wirePrimitiveId = w.getState_PrimitiveId?.();
		if (typeof wirePrimitiveId !== 'string' || !wirePrimitiveId) throw new Error('Missing observed wire primitive identity.');
		const net = w.getState_Net?.() ?? '';
		if (typeof net !== 'string') throw new Error(`Invalid net for wire ${wirePrimitiveId}.`);
		const { segments, rawLine, rawEncoding } = parseObservedWireLine(w.getState_Line());
		return segments.map((seg, segmentIndex) => ({ seg, wirePrimitiveId, net, segmentIndex, rawLine, rawEncoding }));
	});
}

export function wirePointOnSegment(p: WireAnchor, s: WireSegment): boolean {
	const e = WIRE_CONTACT_EPS;
	if (![p.x, p.y, ...s].every(Number.isFinite)) return false;
	if (p.x < Math.min(s[0], s[2]) - e || p.x > Math.max(s[0], s[2]) + e || p.y < Math.min(s[1], s[3]) - e || p.y > Math.max(s[1], s[3]) + e) return false;
	return Math.abs((p.x - s[0]) * (s[3] - s[1]) - (p.y - s[1]) * (s[2] - s[0])) <= e * Math.max(1, Math.hypot(s[2] - s[0], s[3] - s[1]));
}

export function classifyWireContact(a: WireSegment, b: WireSegment): WireContact {
	const e = WIRE_CONTACT_EPS;
	const axis = (s: WireSegment): number => {
		if (!s.every(Number.isFinite)) return -1;
		const h = Math.abs(s[1] - s[3]) <= e, v = Math.abs(s[0] - s[2]) <= e;
		return h === v ? -1 : h ? 0 : 1;
	};
	const aa = axis(a), bb = axis(b);
	if (aa < 0 || bb < 0) return 'unknown';
	if (aa === bb) {
		const fixed = aa === 0 ? 1 : 0, moving = 1 - fixed;
		if (Math.abs(a[fixed] - b[fixed]) > e) return 'disjoint';
		const overlap = Math.min(Math.max(a[moving], a[moving + 2]), Math.max(b[moving], b[moving + 2])) - Math.max(Math.min(a[moving], a[moving + 2]), Math.min(b[moving], b[moving + 2]));
		return overlap < -e ? 'disjoint' : overlap > e ? 'collinear-overlap' : 'endpoint-touch';
	}
	const h = aa === 0 ? a : b, v = aa === 1 ? a : b;
	const p = { x: v[0], y: h[1] };
	if (!wirePointOnSegment(p, a) || !wirePointOnSegment(p, b)) return 'disjoint';
	// Match Go schguard.SegmentsContact's endpoint-on-segment tolerance rather
	// than introducing a second radial endpoint tolerance at a noisy corner.
	const endOn = (s: WireSegment, other: WireSegment) => wirePointOnSegment({x:s[0],y:s[1]}, other) || wirePointOnSegment({x:s[2],y:s[3]}, other);
	return endOn(a,b) || endOn(b,a) ? 'endpoint-touch' : 'proper-cross';
}

/** Return segment indices per physical island. Anchors at a bare X contact both
 * arms, so the non-junction exception is inapplicable. Same primitive IDs never
 * force union: a primitive may contain independent segment records.
 */
export function physicalWireIslands(segments: Array<{ seg: WireSegment }>, anchors: WireAnchor[] = []): number[][] {
	const parent = segments.map((_, i) => i);
	const find = (i: number): number => { while (parent[i] !== i) { parent[i] = parent[parent[i]]; i = parent[i]; } return i; };
	const union = (a: number, b: number) => { parent[find(a)] = find(b); };
	for (let i = 0; i < segments.length; i++) {
		if (classifyWireContact(segments[i].seg, segments[i].seg) === 'unknown') throw new Error('Unknown non-orthogonal or zero-length wire topology.');
		for (let j = i + 1; j < segments.length; j++) {
			const relation = classifyWireContact(segments[i].seg, segments[j].seg);
			if (relation === 'unknown') throw new Error('Unknown wire contact.');
			if (relation === 'endpoint-touch' || relation === 'collinear-overlap' || (relation === 'proper-cross' && anchors.some(p => wirePointOnSegment(p, segments[i].seg) && wirePointOnSegment(p, segments[j].seg)))) union(i, j);
		}
	}
	const groups = new Map<number, number[]>();
	segments.forEach((_, i) => { const k = find(i); const group = groups.get(k) ?? []; group.push(i); groups.set(k, group); });
	return [...groups.values()];
}

/** Legacy element move/disconnect implementations only understand one straight
 * stub per primitive. Refuse richer topology BEFORE their first write. This is
 * a capability boundary, not a guessed conversion of a branch into a polyline.
 */
export function assertLegacySimpleWireOperation(segments: ObservedWireSegment[], selected: Set<string>, delta?: WireAnchor): Map<string, WireSegment> {
	const plans = new Map<string, WireSegment>();
	for (const id of selected) {
		const own = segments.filter(s => s.wirePrimitiveId === id);
		if (own.length === 0) continue; // selected component/marker, not a wire
		if (own.length !== 1 || classifyWireContact(own[0].seg, own[0].seg) === 'unknown') throw new Error(`Wire ${id} is not one simple observed segment; take a fresh raw snapshot and use data-driven sch compose --preserve-instances.`);
		plans.set(id, own[0].seg);
	}
	const moved = (s: ObservedWireSegment): WireSegment => delta && selected.has(s.wirePrimitiveId)
		? [s.seg[0]+delta.x,s.seg[1]+delta.y,s.seg[2]+delta.x,s.seg[3]+delta.y] : s.seg;
	for (let i=0;i<segments.length;i++) for(let j=i+1;j<segments.length;j++) {
		if (!selected.has(segments[i].wirePrimitiveId) && !selected.has(segments[j].wirePrimitiveId)) continue;
		const relations = [classifyWireContact(segments[i].seg,segments[j].seg)];
		if(delta) relations.push(classifyWireContact(moved(segments[i]),moved(segments[j])));
		if(relations.some(r=>r==='proper-cross'||r==='unknown')) throw new Error('Crossing/unknown wire topology is not supported by legacy move/disconnect; take a fresh raw snapshot and use data-driven sch compose --preserve-instances.');
	}
	return plans;
}
