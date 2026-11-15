type Listener = () => void

export interface SelectionStore {
  readonly selected: Set<string>
  readonly count: number
  toggle: (id: string) => void
  selectAllLoaded: (ids: string[]) => void
  selectRange: (fromIndex: number, toIndex: number, loadedIds: string[]) => void
  selectNone: () => void
  prune: (loadedIds: string[]) => { pruned: string[]; remaining: Set<string> }
  subscribe: (listener: Listener) => () => void
}

export function createSelectionStore(): SelectionStore {
  let selected = new Set<string>()
  const listeners = new Set<Listener>()

  const emit = () => {
    for (const listener of listeners) {
      listener()
    }
  }

  return {
    get selected() {
      return selected
    },
    get count() {
      return selected.size
    },
    toggle: (id: string) => {
      const next = new Set(selected)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      selected = next
      emit()
    },
    selectAllLoaded: (ids: string[]) => {
      selected = new Set(ids)
      emit()
    },
    selectRange: (fromIndex: number, toIndex: number, loadedIds: string[]) => {
      const next = new Set(selected)
      const start = Math.min(fromIndex, toIndex)
      const end = Math.max(fromIndex, toIndex)
      for (let i = start; i <= end; i++) {
        next.add(loadedIds[i])
      }
      selected = next
      emit()
    },
    selectNone: () => {
      selected = new Set<string>()
      emit()
    },
    prune: (loadedIds: string[]) => {
      const loadedSet = new Set(loadedIds)
      const pruned: string[] = []
      const remaining = new Set<string>()

      for (const id of selected) {
        if (loadedSet.has(id)) {
          remaining.add(id)
        } else {
          pruned.push(id)
        }
      }

      selected = remaining
      emit()
      return { pruned, remaining }
    },
    subscribe: (listener: Listener) => {
      listeners.add(listener)
      return () => {
        listeners.delete(listener)
      }
    },
  }
}
