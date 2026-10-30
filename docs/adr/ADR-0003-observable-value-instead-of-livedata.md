# ADR-0003: livedata 대신 Go 안에 작은 구독 구조를 둔다

## Status

Accepted

## Date

2026-10-25

## Context

원본 MonoSketch 는 모듈 사이의 변경 알림을 `livedata`(Android LiveData 와 비슷한 구조)로 한다.
`livedata` 는 `lifecycle` 에 묶여 있고, throttle 은 브라우저 타이머(`setTimeout`, `requestAnimationFrame`)를 쓴다.

UI 가 아닌 모듈이 쓰는 기능은 다음과 같다.

- `shape`, `shape-selection`: 값 읽기, 값 설정, 구독
- `statemanager`, `action-manager`: 위 기능에 더해 `distinctUntilChange`, `combineLiveData`, `map`, 다음 프레임으로 미루는 throttle

검토한 방식은 다음과 같다.

- 버전 번호로 당겨 읽기: Go 는 값과 버전만 갖고, JS 가 이벤트를 넘긴 뒤 바뀐 상태를 한 번에 받아 간다. 코드는 가장 적지만 원본 구조와 많이 달라진다.
- Go 안에 작은 구독 구조: 원본 구조와 1:1 로 맞출 수 있다. 대신 구독 코드가 늘어난다.
- Go channel: WASM 은 스레드가 하나라서 goroutine 과 동기화 비용만 늘어날 수 있다.

## Decision

**Go 안에 작은 구독 구조(`internal/observable`)를 둔다.**

- `observable.Value[T]` 는 값 읽기(`Value`), 값 설정(`Set`), 구독(`Observe`)을 제공한다.
- `Observe` 는 구독하는 즉시 현재 값을 한 번 전달한다. (원본과 같다)
- lifecycle 은 옮기지 않는다. `Observe` 가 돌려주는 해제 함수를 호출해 구독을 끝낸다.
- `distinctUntilChange`, `combineLiveData`, `map`, throttle 은 `statemanager` 를 옮길 때 필요한 만큼만 추가한다.

## Consequences

- 원본 코드를 옮길 때 구조를 거의 그대로 따를 수 있다.
- 구독 해제를 잊으면 구독자가 계속 남는다. 원본에서는 lifecycle 이 하던 일이다.
- 다음 프레임 throttle 은 브라우저 타이머가 필요하다. ADR-0006 에서 JS 가 맡기로 했으므로 `observable` 에는 넣지 않는다.
- JS 쪽 UI 가 Go 의 변경을 받는 방식은 ADR-0006 에서 정했다.
