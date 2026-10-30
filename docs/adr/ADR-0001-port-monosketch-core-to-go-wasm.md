# ADR-0001: MonoSketch 를 Go WASM 으로 포팅한다

## Status

Accepted

## Date

2026-10-25

## Context

[MonoSketch](https://github.com/tuanchauict/MonoSketch) 는 브라우저에서만 동작하는 ASCII 다이어그램 편집기다.
Kotlin/JS 와 Compose for Web 으로 작성되어 있고, main 소스는 약 16k 줄이다. (Apache 2.0)

원본 모듈은 성격에 따라 세 묶음으로 나뉜다.

| 묶음 | 모듈 | 규모 | 브라우저 의존 |
|---|---|---|---|
| 핵심 로직 | `graphicsgeo`, `monobitmap`, `monobitmap-manager`, `monoboard`, `shape`, `shape-serialization`, `shapesearcher`, `shape-interaction-bound`, `shape-selection`, `shape-clipboard` | 약 6k | 없음 (`shape-clipboard` 만 클립보드 접근) |
| 상태·명령 | `statemanager`, `action-manager`, `keycommand`, `livedata`, `lifecycle`, `commons`, `uuid` | 약 3.5k | 일부 |
| UI·저장 | `ui-canvas`, `ui-toolbar`, `ui-modal`, `ui-compose-ext`, `ui-theme`, `ui-app-state-manager`, `html-dsl`, `export-shapes-modal`, `browser-manager`, `store-dao`, `store-manager` | 약 6k | 강함 (Compose, Canvas, localStorage) |

원본 `statemanager` 는 `ui-canvas`, `ui-modal`, `ui-toolbar` 에 직접 의존한다. 그래서 상태 로직과 UI 가 서로 얽혀 있다.

검토한 실행 방식은 다음과 같다.

- TUI (bubbletea): 결과물이 ASCII 라서 잘 맞지만, 터미널 크기와 폰트(박스 문자 폭)에 제약을 받는다.
- Go 서버 + 웹: 마우스를 움직일 때마다 서버와 주고받아야 해서 반응 속도가 나쁘다.
- Go WASM + 웹 UI: 원본처럼 모든 계산을 클라이언트에서 한다. `~/src/bluemir/grim` 에서 이미 쓰고 있는 방식이다.

## Decision

**핵심 로직과 상태·명령은 Go 로 옮겨 WASM 으로 빌드한다. UI 는 lit-html 과 canvas 로 새로 만든다.**

- 포팅 기준은 원본 Kotlin main 브랜치다. (TypeScript 재작성 중인 `port-to-js` 브랜치는 기준으로 삼지 않는다.)
- 원본 기능 전체를 옮긴다.
- 저장 포맷은 원본과 호환하지 않고 새로 정의한다. (ADR-0002)
- Go 와 JS 의 경계
	- Go(WASM) 가 도형 트리, 선택, undo/redo, 마우스·키 명령 해석, 문자 보드 계산을 모두 맡는다.
	- JS 는 DOM 이벤트를 WASM 에 넘기고, WASM 이 돌려준 결과(문자 보드, 선택 영역, 툴바 상태)를 그린다.
	- 원본처럼 상태 로직이 UI 에 직접 의존하지 않게 끊는다.
- 페이지 구성 (AGENTS.md 의 "SPA 금지" 해석)
	- 페이지마다 서버가 따로 렌더링한다. 지금 페이지는 로컬 편집(`/edit`)과 서버 문서(`/p/<id>`)다.
	- 클라이언트 라우팅은 쓰지 않는다.
	- 문서 페이지 하나만 상호작용이 많은 페이지로 둔다. 권한에 따라 편집 화면이나 읽기 전용 화면으로 열린다.
- 빌드는 grim 의 `scripts/makefile.d/wasm.mk` 방식(`GOOS=js GOARCH=wasm`, `wasm_exec.js` 복사)을 따른다.
- 템플릿에 있는 로그인(webauthn), admin, posts, gRPC 는 건드리지 않는다.

## Consequences

- 핵심 로직은 순수 Go 패키지가 되므로 `go test` 로 테스트할 수 있다. 원본 테스트(`shape`, `monoboard`, `monobitmap-manager`, `livedata`)도 함께 옮긴다.
- UI 약 6k 줄은 옮기는 것이 아니라 새로 만드는 일이다. 작업량이 가장 크다.
- WASM 파일 크기가 커지고 첫 로딩이 느려진다. (Go WASM 은 보통 수 MB)
- JS 와 WASM 경계를 오가는 비용이 생긴다. 마우스 이동처럼 자주 생기는 이벤트는 호출 횟수와 넘기는 데이터 크기를 줄여야 한다.
- 원본 저장 파일은 읽을 수 없다.

## 아직 정하지 않은 것

- `livedata`(Android 식 반응형 값)를 Go 에서 어떤 방식으로 대신할지: ADR-0003 에서 정함
- 문자 보드를 canvas 에 그릴 때 매번 전체를 넘길지, 바뀐 영역만 넘길지: ADR-0006 에서 정함
