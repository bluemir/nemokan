# ADR-0009: 편집기 글꼴로 D2Coding 원본 TTF 를 내장한다

## Status

Accepted

## Date

2026-10-25

## Context

원본 MonoSketch 는 JetBrains Mono 를 쓴다. 지금은 시스템 monospace 글꼴로 그리는데, 다음 문제가 있다.

- 사용자 환경마다 글꼴이 달라 칸 너비와 모양이 달라진다.
- 한글은 시스템 대체 글꼴로 그려져 폭이 정확히 두 칸이 아니다 (ADR-0008).

D2Coding(NAVER)은 고정폭 글꼴이고, 한글 폭이 영문 폭의 정확히 두 배다. SIL Open Font License 1.1 이고, Reserved Font Name 은 `D2Coding`, `D2Coding-Bold` 다.

OFL 에서 Reserved Font Name 이 있으면, 수정한 글꼴(형식 변환이나 일부 글자만 남기는 subset 포함)은 그 이름을 쓸 수 없다.

검토한 방식은 다음과 같다.

- 원본 TTF 를 그대로 배포: 약 4MB. 라이선스 문제가 없다.
- WOFF2 로 변환하거나 subset: 크기가 줄지만 수정판이므로 글꼴 이름을 바꿔야 한다.
- CDN 에서 불러오기: 저장소와 바이너리가 가볍지만 외부 서비스에 의존한다.

## Decision

**D2Coding 1.4.0 regular TTF 를 수정하지 않고 저장소(`third_party/d2coding/`)에 두고, 빌드할 때 `assets/bundle/d2coding/` 로 복사해 서빙한다.**

- prod 빌드에서는 `bundle/*` 와 함께 바이너리에 embed 된다.
- 라이선스 전문은 `third_party/d2coding/OFL.txt`, 고지는 `NOTICE` 에 둔다.
- canvas 는 굵은 글씨를 쓰지 않으므로 bold 는 넣지 않는다.
- `@font-face` 는 shadow root 안에서 동작하지 않으므로 편집 페이지 문서에 둔다.
- JS 는 글꼴이 로드된 뒤 칸 크기를 잰다.

## Consequences

- 편집 페이지를 처음 열 때 약 4MB 를 더 받는다. 정적 파일은 revision 경로로 캐시된다.
- 크기를 줄이려면 이름을 바꾼 subset 글꼴을 만들어야 한다. 필요해지면 다시 정한다.
