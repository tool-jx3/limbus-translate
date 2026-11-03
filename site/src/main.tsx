import { render } from 'preact'
import './styles.css'

function App() {
  return (
    <main class="boot">
      <h1>Limbus 譯文對照</h1>
      <p>KR、JP、EN 與 TW 譯文的對照檢視。</p>
    </main>
  )
}

const root = document.getElementById('app')
if (root) render(<App />, root)
