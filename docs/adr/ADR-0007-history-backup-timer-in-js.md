# ADR-0007: 자동 백업 타이머와 저장은 JS 가 맡고, Go 는 undo 기록과 저장할 내용을 만든다

## Status

Accepted

## Date

2026-10-25

## Context

원본 `StateHistoryManager` 는 다음처럼 동작한다.

- 편집 중이 아닐 때 도형이 바뀌면 300ms 뒤 백업 타이머를 건다.
- 타이머가 돌 때 그 사이 바뀐 것이 없으면, 현재 상태를 undo 스택에 넣고 로컬 저장소에 쓴다.
- 원본 로컬 저장소는 동기 API 인 localStorage 다.

ADR-0002 에서 로컬 저장은 IndexedDB 로 하기로 했고, IndexedDB 는 비동기 API 다.

검토한 방식은 다음과 같다.

- JS 타이머 + JS 저장: Go 는 백업할 변경이 생겼다고 알리고, JS 가 300ms 뒤 Go 의 백업 함수를 부른다. Go 는 undo 스택에 넣고 저장할 내용을 돌려준다. JS 가 IndexedDB 에 쓴다.
- Go 타이머 + JS 저장: Go 가 `time.AfterFunc` 로 시간을 잰다. WASM 에서도 동작하지만 Go 고루틴이 JS 이벤트 사이에서 상태를 건드리게 되어 동시성을 따로 다뤄야 한다.
- Go 타이머 + Go 저장: Go 가 `syscall/js` 로 IndexedDB 를 직접 다룬다. 비동기 콜백을 Go 에서 처리해야 해서 코드가 가장 많다.

## Decision

**JS 가 300ms 타이머와 IndexedDB 쓰기를 맡는다. Go 는 백업할 변경을 알리고, 백업 함수가 불리면 undo 스택에 넣고 저장할 내용을 돌려준다.**

```
Go: 도형 변경 (편집 중 아님) → JS 콜백 ("backup")
JS: 마지막 알림 뒤 300ms 지나면 → Backup() 호출 → 돌려받은 내용을 IndexedDB 에 씀
Go: Backup() 에서 알림 뒤 바뀐 것이 없으면 undo 스택에 넣고 저장할 내용을 돌려줌
```

- 시작할 때는 JS 가 IndexedDB 에서 읽은 내용을 Go 에 넘겨 복원한다.
- undo 스택은 Go 메모리에만 둔다 (원본과 같다).

## Consequences

- Go 쪽 상태 코드에 타이머와 고루틴이 없어 JS 없이 테스트할 수 있다.
- 저장할 내용의 모양(문서 이름, 화면 위치 포함 여부)은 로컬 저장을 만들 때 정한다.
