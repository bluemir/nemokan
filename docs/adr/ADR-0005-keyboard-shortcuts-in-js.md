# ADR-0005: 단축키 표는 JS 에 두고 Go 는 동작만 받는다

## Status

Accepted

## Date

2026-10-25

## Context

원본에서 키 입력은 다음 순서로 처리된다.

```
keydown → KeyCommandController (텍스트 입력 중이면 무시, 브라우저 기본 동작 막기)
        → KeyCommand (keyCode, cmd, shift → 명령)
        → ActionManager (명령 → 동작)
        → statemanager (동작 실행)
```

툴바와 메뉴는 키를 거치지 않고 `ActionManager` 에 동작을 바로 넘긴다. 단축키 도움말 dialog 는 `KeyCommand` 를 쓰지 않고 키 이름을 UI 코드에 따로 적는다. Shift 를 누르고 있는지는 canvas 쪽 UI 가 쓴다.

어느 방법이든 툴바가 Go 에 동작을 넘기는 입구는 필요하다. 차이는 단축키 표를 어디에 두느냐다.

- Go 에서 변환: JS 는 키 이벤트를 그대로 넘긴다. 단축키 규칙을 Go 에서 테스트할 수 있지만, Go 로 들어오는 입구가 키와 동작 두 개가 된다.
- JS 에서 키를 동작으로 변환: 단축키 표와 도움말 dialog 가 JS 한곳에 모인다. Go 로 들어오는 입구는 동작 하나다. 단축키 표는 Go 에서 테스트하지 못한다.

단축키 표는 어느 키가 어느 동작인지만 담고 있어 테스트할 로직이 거의 없다.

## Decision

**단축키 표는 JS 에 두고, JS 가 키를 동작으로 바꿔 Go 에 넘긴다.**

- 키는 `event.code` 로 판별한다. 원본의 `keyCode` 는 폐기되었고, `event.key` 는 한글 입력 상태에서 `ㄱ` 같은 글자가 와서 단축키가 동작하지 않는다.
- 텍스트 입력 중인지, 브라우저 기본 동작을 막을지도 JS 가 정한다.
- Shift 상태는 JS 의 canvas 쪽에서 쓴다.
- 원본 `keycommand` 모듈은 Go 로 옮기지 않는다.
- Go 의 `action` 패키지가 동작 타입을 정의한다. JS 와 Go 사이에서 동작은 `"type"` 으로 구분하는 JSON 으로 주고받는다 (ADR-0004 와 같은 방식).

```json
{"type":"moveShapes","offsetRow":0,"offsetColumn":-1}
```

## Consequences

- 동작의 `type` 이름과 필드가 Go 와 JS 사이의 약속이 된다. 바꿀 때는 양쪽을 함께 바꾼다.
- 단축키 표와 도움말 dialog 를 같은 데이터로 그릴 수 있다.
- 단축키 규칙은 JS 쪽에서 확인한다.
