// The Web Worker of the full-text search: it takes a query from the page and
// answers it. The matching is in engine.ts.
import { SearchEngine, type Query } from './engine'

const engine = new SearchEngine()

self.addEventListener('message', (event: MessageEvent<Query>) => {
  void engine.run(event.data).then((reply) => self.postMessage(reply))
})
