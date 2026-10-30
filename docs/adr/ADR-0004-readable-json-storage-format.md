# ADR-0004: 저장 포맷은 읽기 쉬운 새 JSON 으로 한다

## Status

Accepted

## Date

2026-10-25

## Context

원본 MonoSketch 는 도형을 kotlinx.serialization JSON 으로 저장한다. 필드 이름을 짧게 줄이고(`i`, `v`, `b`, `e`), 값은 `|` 로 이어 붙인 문자열로 넣는다.

```json
{"type":"R","i":"02-...","v":123,"b":"0|0|5|3","e":{"fe":false,"fu":"F1","be":true,"bu":"S1","du":"1|0|0","rc":"NNNN"}}
```

ADR-0002 에서 원본 포맷과 호환하지 않기로 했으므로 포맷을 새로 정할 수 있다.

검토한 방식은 다음과 같다.

- 원본과 같은 모양: 파일이 작고, 나중에 원본 파일을 읽는 기능을 넣기 쉽다. 대신 사람이 읽기 어렵고, 서버에서 내용을 들여다보기 어렵다.
- 읽기 쉬운 새 JSON: 파일이 조금 커지지만 디버깅과 서버 쪽 처리가 쉽다.

## Decision

**필드 이름을 풀어 쓴 새 JSON 으로 저장한다.**

- 도형 종류는 `"type"` 으로 구분한다: `rectangle`, `text`, `line`, `group`.
- 좌표와 크기는 객체로 쓴다. Rect 는 `{"left", "top", "width", "height"}` 로 펼친다.
- 방향, 정렬, 선의 끝은 문자열로 쓴다: `"horizontal"`, `"middle"`, `"start"`.
- 스타일은 원본처럼 id 만 저장한다. 모르는 id 는 그때의 기본 스타일로 복원한다.

```json
{"type":"rectangle","id":"02-...","versionCode":123,"bound":{"left":0,"top":0,"width":5,"height":3},
 "extra":{"isFillEnabled":false,"fillStyleId":"F1","isBorderEnabled":true,"borderStyleId":"S1",
          "dashPattern":{"dash":1,"gap":0,"offset":0},
          "corner":{"leftTop":false,"leftBottom":false,"rightTop":false,"rightBottom":false}}}
```

### 파일 전체

문서 하나를 담는 파일은 다음과 같다.

```json
{"version":1,"name":"diagram","offset":{"left":0,"top":0},
 "root":{"id":"02-...","versionCode":123,"shapes":[...]},
 "connectors":[{"lineId":"...","anchor":"start","targetId":"...","ratio":{"left":0.5,"top":0},"offset":{"left":0,"top":0}}],
 "modifiedTimestampMillis":1792900000000}
```

- 원본은 `name`, `offset` 을 `extra` 객체로 한 번 더 감쌌지만, 담는 것이 둘뿐이라 펼쳤다.
- `root` 는 언제나 group 이므로 `"type"` 을 넣지 않는다.
- version 은 1 부터 시작한다. 원본의 version 0, 1, 2 는 호환하지 않으므로 읽지 않는다.
- 모르는 version 의 파일은 읽지 않고 오류로 돌려준다.

## Consequences

- 원본 파일을 읽으려면 별도 변환기가 필요하다.
- 포맷이 바뀌면 version 을 올리고, 이전 version 을 읽는 변환을 그때 넣는다.
