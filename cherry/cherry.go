// Package cherry is a from-scratch terminal UI framework.
//
// Design goals, in order:
//
//  1. Components own their state and their input handling. There is no global
//     message bus, no Update(Msg) plumbing, no model/view ceremony.
//  2. Declarative layout: a flex solver assigns rectangles; widgets never do
//     manual width math.
//  3. A retained cell grid with dirty-row diffing drives output, so idle
//     frames cost nothing and partial updates touch only changed cells.
//
// Layering (import direction, strictly downward):
//
//	term -> input -> cell -> render -> layout -> widget -> app (root)
//
// Nothing outside cherry imports term/input directly except through app.
package cherry

// Version is the semantic version of the cherry framework.
const Version = "0.1.0"
