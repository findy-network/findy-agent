package presentproof

import (
	"encoding/base64"
	"reflect"
	"testing"

	"github.com/findy-network/findy-agent/agent/aries"
	"github.com/findy-network/findy-agent/agent/didcomm"
	"github.com/findy-network/findy-agent/agent/pltype"
	"github.com/findy-network/findy-agent/std/decorator"
	"github.com/findy-network/findy-common-go/dto"

	"github.com/lainio/err2/assert"
)

var request = `{
    "@type": "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/present-proof/1.0/request-presentation",
    "@id": "b1220020-bfd6-408e-a51d-3cfcd2a99acb",
    "request_presentations~attach": [
      {
        "@id": "a1f23394-df26-4cd4-8afb-8068244ca7f9",
        "mime-type": "application/json",
        "data": {
          "base64": "eyJuYW1lIjogIlByb29mIG9mIEVkdWNhdGlvbiIsICJ2ZXJzaW9uIjogIjEuMCIsICJub25jZSI6ICI4MTIxNjIxNDkzMTMyMDgyMzUxODgwNjA3MTI5MTcwNzY5MTgwNiIsICJyZXF1ZXN0ZWRfYXR0cmlidXRlcyI6IHsiMF9uYW1lX3V1aWQiOiB7Im5hbWUiOiAibmFtZSIsICJyZXN0cmljdGlvbnMiOiBbeyJpc3N1ZXJfZGlkIjogIlNhemdWcmVVWHR3RjRaQndBeFVQd1UifV19LCAiMF9kYXRlX3V1aWQiOiB7Im5hbWUiOiAiZGF0ZSIsICJyZXN0cmljdGlvbnMiOiBbeyJpc3N1ZXJfZGlkIjogIlNhemdWcmVVWHR3RjRaQndBeFVQd1UifV19LCAiMF9kZWdyZWVfdXVpZCI6IHsibmFtZSI6ICJkZWdyZWUiLCAicmVzdHJpY3Rpb25zIjogW3siaXNzdWVyX2RpZCI6ICJTYXpnVnJlVVh0d0Y0WkJ3QXhVUHdVIn1dfSwgIjBfc2VsZl9hdHRlc3RlZF90aGluZ191dWlkIjogeyJuYW1lIjogInNlbGZfYXR0ZXN0ZWRfdGhpbmcifX0sICJyZXF1ZXN0ZWRfcHJlZGljYXRlcyI6IHsiMF9hZ2VfR0VfdXVpZCI6IHsibmFtZSI6ICJhZ2UiLCAicF90eXBlIjogIj49IiwgInBfdmFsdWUiOiAxOCwgInJlc3RyaWN0aW9ucyI6IFt7Imlzc3Vlcl9kaWQiOiAiU2F6Z1ZyZVVYdHdGNFpCd0F4VVB3VSJ9XX19fQ=="
        }
      }
    ]
  }`

var presentation = `{
    "@type": "did:sov:BzCbsNYhMrjHiqZDTUASHg;spec/present-proof/1.0/presentation",
    "@id": "96a5893d-113a-435d-8637-3f33ebebc620",
    "~thread": { "thid": "b1220020-bfd6-408e-a51d-3cfcd2a99acb" },
    "presentations~attach": [
      {
        "@id": "libindy-presentation-0",
        "mime-type": "application/json",
        "data": {
          "base64": "eyJwcm9vZiI6IHsicHJvb2ZzIjogW3sicHJpbWFyeV9wcm9vZiI6IHsiZXFfcHJvb2YiOiB7InJldmVhbGVkX2F0dHJzIjogeyJkYXRlIjogIjIzNDAyNjM3NDIzODc2MzI0MDk4MjU2NTE5MzE3Njk1NDMzMTk2ODEzMjE3Nzg1Nzk1MzE3MjIwNjgwNDE1ODEyMzQ4ODAxMDg2NTg2IiwgImRlZ3JlZSI6ICI0NjAyNzMyMjkyMjA0MDg1NDIxNzg3MjkzMjg5NDg1NDgyMzUxMzI5MDUzOTM0MDAwMDE1ODIzNDI5NDQxNDc4MTM5ODQ2NjA3NzIiLCAibmFtZSI6ICI2MjgxNjgxMDIyNjkzNjY1NDc5Nzc3OTcwNTAwMDc3Mjk2ODA1ODI4Mzc4MDEyNDMwOTA3NzA0OTY4MTczNDgzNTc5NjMzMjcwNDQxMyJ9LCAiYV9wcmltZSI6ICI4ODM5MTA1Mzg3NzI4MTQ3MzQ5NTYzMTQ5MzIyNTM5NDQ5NjkzNzg5OTU5OTg3NzUwMjIzODE3MzUxODEzOTU1ODAzMjgzOTc1NjkzNjM5Njc2NjU3NTQ4ODQyMzk1NDI3OTAwOTY1NTc3MDQ2NTc0MzUxOTE2MjAwNTk2MTI0NTM4NzgzMDg1ODI1MTY5NzEzMDA0MjA1MDQwMzkwMzMzMjg4NzEzMDU5NjI3NDUyNzA2OTAyNzM0ODMzNjA2MjAyMjUzODczMTUwMDAzNDg4NDc0MTgzNDg4MjE5NzI5NjE2MDk2MDYzODU5NjIyNzY3MDIxMDEwMTExMTgzNzEyMDg2ODA5Mjg4MTAyMzM4Mjc3NjgzODAxNjM3MDg5NjY2NDEyMjk4NjMzMjQ2OTY3OTIwOTUyNDIxMDQyNTIxODYzMzY2MDc0NDgzMzk3NzUwMDczNjU0NzM0NDA1MDA2MTkzNzQ1NTY2MzczMTI1ODAwMjMwNTQ0NTU1NzMxMTMxNjQ2MTQyMzE3NzUxODY0MTcxMDM5MjQzNTgyMTQ0NTQzNjk1MTEyNDM3NTU4MTU1NzEwMzEwMDI3OTI4NzIxNjA0MTQ5NjUzMTQ5OTk0Njc4MzI5NzAxMjM2NDk1MTQ5MDEwNzg1OTc3MDEwOTQ4MTcwODMwNTExNTA3ODQyOTEwOTE1NDc2MDExODkwMTIyOTE0MjgxMDgzOTE3NDk2NjgyODE3ODY0NTkwNzY5NjAyOTc5Mzk3MDk1ODA3MTI5OTA5MDEyOTAzOTA5NzM3NDUxMTMwMzgyNDQ3MTQ3MjA0OTQ5NDU5MyIsICJlIjogIjI5NDAyODA3ODQxMzcxMDM3NzcxODg4MzQyMzA0MzA0MDEzNDcxNjkzOTE5NzcyMzE1ODc0MTk3MzkxNDA3NzU1NzIyNzc1MjU5MDc0MDg2NDUzMzk3NjcxNjY1OTM1MjkwNjM4NDkwMjUwMTk4MTIwMzI0NzU0NDkzNDczMDQ3MzM1MDQwOTA2IiwgInYiOiAiNjc2OTI0MDY0NDIxMDQ2MDgxNDA2MjgyMTkyMzU2MDQxNjk1NTIwNTIxMDg2Mjk2MDExMTExMTg0NjI5NzU1NTMxNjg5NjE0Mzg5MTUwNDY4OTYxMjc5OTAzNjk3MDk1MzE0NjIxNDY0NzM0NTEyNDE1NzQ2MzUxOTY3NDk0MTU3NzM1MDQxMTU2NzQxMDE1NTYzNDAyMzc2MDExNzc5MzI1Mjc5MzY1NjE5MjA2NzkyMTkzNjQ2NzAxOTE2MzYyNTIwNzMyODgxNDczMzIwMDkzNjM4ODA3MzY4NzUwMjc2MDU3OTQ4MTUxMDAyMzM0OTQ0ODEzMzY1OTMwMjkwMjc4OTE1MDQ2ODExNDExOTk5NzQ4MzIwNTEzNDc3NzQ2MTE5MDQ3NzA2NjQ2ODc5NTE3NzgwMDIwMjA1NTY1ODQ2MTc4Mjg5ODY0NTMxNDc0MTIwNjk3MTY2MDExMTc2ODI3NDMzMTQ3MjY0NjY4NTUxOTI3MzY0NDAzNjk5MTUyNTM4NzY4OTQ5ODU3MTE1NjEzMDExMTY1MzA2OTQxOTIxOTkyNDgwODU1OTQxOTUwNjQzMjI4NzIzNzI0NTUxMzc2NDM2NTM3MjM2Njg1ODU3NzExNDQwMTYyMzU5MjczMjU5MzIyMjkzNjM0NDY5NDk3ODY4Nzk0NDM0NTU1NTE0ODcyNzY0Nzk3ODk4MTkwNDYzMDE5NDYzOTY4MjkwNjk2MjYxNDQ1MTkzNTEzNjQ1MzExNzM0NjMwMzMyMTg0ODAyNjI3MTk0MTQ1MDE0Mzk0NTU4MzgxMDg2MDEzODMwNzU4NzU0ODIyNzIyNDAxMjkxMDg1NDA3NTc4MjkyNzgwNzEzNjgzNTE2OTc4NDUyOTQyNDU5ODI3ODY0ODY4NTU0OTI2Njg0ODk0NjIyMTM5MTcyNjgzMDc5MjM2ODY3OTgyMjExMTQ3NTQ3MDc4MDkzMDI5NzgwMTU5OTI4NTcwMzUwMzYyOTE4ODQ0NTQ3MDAzNjMyNzc0MzYzNjA3ODkyODExNTMwNTQ3MzA4NjQ3MTUzNzgyNDQ3ODc2ODM1NTM3NzM2MDYwMzQ4MzM5OTA4MTkzNjQ1MzkzODI0NjI1Mzk4MjAwODU5NDQxMTMyMDI1NzM2NDkzMjI4MzA2MDQ4MTQwNTI3NTUzMjEzNDIzNDg3NTAwNjI0Njg1MDE2NjU5MjAzMDA4OTExMDU5NzkyMDg1NjE3IiwgIm0iOiB7ImFnZSI6ICIzMjQwNjExOTc2Njc0MjUwNzgzODQ5NzUxMTgxODQ5MzEzOTczNjY4OTcyMTY3MTg5NjQyMzk2MDEwNzc0MTY2NzE4OTc5NDc5MzkyMTA1Mzk1ODYzMTI5OTMwOTgzMTgyNzE0NTUxODc2OTM2NTIyNzUyMTc4MzAyMDY0NzQwNDc1Njc3NzA0NjgyMTM2MTIzMDc5MzU1MjYxMzUzMTk5MTcwNDYzNzQxNjIxODAxMzU4IiwgIm1hc3Rlcl9zZWNyZXQiOiAiNzY4NTQ4NTg4NDY5NjM2Nzc2NDM1OTMyMjI1MzcwNTgyNzQ3MzE1MTgwMzY1NjM2NDEzNDg0MDYwODI2MjE0ODY0ODgwMjUyNzM3MTUzNDYyNzc3MjIxMzI2NzEzODU4OTQ3Mzg2MTQyNzc1MzUxNDYwMTUzOTc4ODUzOTE4NDYyMDc4MjQyMTU4NzkyOTkyNTgxNjYwMDEzMTcxMzY3MzE4MTQwMDc0NTAxMjY1NjgyNyJ9LCAibTIiOiAiMjAwNzU0MDUyMzU0OTczMzMwMzA0Mjc4Nzg5MTk3MTE0NTI1OTM5Mzk4NzEwNDIxODExMzE1MzE5NzY3Mjc4NTEzODg2OTg0MTAxNzEwNDU1Njc2NTU0Mjg2ODczMTc4NDYzMzcxNjEyMTY1Nzg5MjUzNDU0NDUwNTEzMDQ3MDQxNjQ4NzI2NDI0NTU5ODk1MTEzNTM1MDc4NDQzOTU3MDk0Mzk4NDcyNzA4NzczNjIzMyJ9LCAiZ2VfcHJvb2ZzIjogW3sidSI6IHsiMCI6ICI5MjQ1ODU1NjU3MDU4ODg5NTkwOTY0NTU2NDg4MDUwMDg4NDY3NzI1MTE0NTMxNTcwNzUyNzY3NjE0NTYxMzY2NjQxODI2NTQ2MzcwMjc5MTcxMDk2Mzc5NzgwMTU0MzIzNTAyMTQ5NTk0MDY5ODgwNDY1OTUxNjUxMDE1OTM5MzE0NDY0MzQ5MDY4Mjc4Mjg4NjQyODI2MjM1MTc4Mjg4NTMxMDM3MDEwNjM3NDkwNjEyIiwgIjMiOiAiMTMzNzkxNzk0NjcwMDg0MDg3NzI5MzQ2MDI3MTAxMTM5MTcyNjA4MDU2NzM5NDY1ODY2MTM4MTQwOTA0OTk4MzM3MDYyNjQ3MzAyMTE1MTA4MTQxOTc3MDI3MzYyNTU5MTc4NzIxMTUzNjE1NDU1MzY3NDExMzA0NjcxMTc0OTk1OTYwOTU3ODU0NDUwNjMzMTA0MDQ2NDE0NTYzMTM3ODUyMTU2OTIyNDg2OTA5NjQ5MDciLCAiMiI6ICIxMzM2NzE5ODI3MDg4MzYwMzA3MzI4ODg4MTI5MDcxMjg2MzM2NjMyNjU3MjE5OTI3Mjg3Njg4ODI0Nzk1OTQ5MzQ5ODUxNzQwMzQ0NjczNjU5Mjk4NzUyMzczNDQyOTIwMTI4NjQ4NDE4NzIwOTUwOTU4NDgxMzYyODgxODkxNTM1OTk3MTA3MTQ3NzA1NDMzOTYwMDMxNDU3MTQ1NzA4MzMyNzM2MTQzOTUzNDYwOTU1MSIsICIxIjogIjE1Mjk1NTgzODI0MTgxMzgyNDE1MjUxMzAxOTA4NDY2OTA3MDExNDQ2MzY3MTgwNDkzNzc5MDAxMjY5MjI5OTIwNjU4Nzg4Mzk0NjE1NzY4MjA2OTUyNDM2MDk5NTE5MjM0NjY3NjU2MjM1NTYwOTkxMjc0ODY0NzkyMjUwOTAzOTkzMTg2MjkxNTI0NTE4MzcyODk0MjY0MDgwMTUzNTkyMjc0NzM4MzI5MjE5OTI5OTc3In0sICJyIjogeyIyIjogIjE2MDI2NTc3MTUyMDkwODMzNjg5NTYyNjI4NjgzODkyNzQ4MDczMTU0OTkwOTcxMjIxNTIzNjYyOTE4MjA0NTI1NTg3MDkzNTg3OTAxNjE4NjQyMjMzNjU1NDk2NDU0MDY5MDIwODQ3ODY3NTE4NzQ2NDI3NTQzMDM5ODMyMjY1NDIyNzIwNDgwNzY2NDUzMzM0NDA0MjI1NDQ5MjU1NjE3MzAwNTE4OTcxNTMyOTQzMjUxODY1MDg3NjQ1MTE4NDE3MjI5NDAwMjEwMDM4MTE3NjE2NjMxNDQxNjE5OTE5MDE5MzM4NjM1NTcyNzM2MzA4ODg2NzA5MDU4MjQ5NjUyMjY5Njg0NDg4OTA3NjU1NzQ1NDg1MTgzNjYzNzE3MzUzOTk1NTkwNTg2Njk1MzMwOTM4NjM1NTk4OTE3OTkxMDk0ODM2NjcyNDM3NjcxMzIzMzYyNDYwODAwMTA0MzY1MjI3MDA3ODY0MzkwNDM0MzU4NzQ0NTM1NTYyMjc4NzI0ODM3MTkyODcwMDY3MjgyNzE5NTU4NTUxMjMxMzg3MzYwODc2OTg0NTIyMjI0Njg5NjMwMTYwNDk4NTQwMzcyMTk4OTgxNjAwMjgwNDEzMDMwODcwOTc1OTEyODAzMjk0NDI1ODU5MjYyMTI2Njg3OTcwMDg0MDQ0ODIxMjE4NDg3MTAwNTc1MjkxODQ2NzE0ODg0NzczNzg0MDcwMjQxNzEwNTg3Njc3NDkyOTYyNTgzNzczNTU1NjM4MTIyMzE1Mzc3MjczMjEyMDMxNzQ3ODcxMTYxMTkxOTEwNjE0MDk4NDM0MjMzNjQ1MTEzMDAyNTk5MzQ5MjQzMTEwNzI1NzE1MzIxNjA1MDQ5MTU3ODY2MTc1MDk3OTQ5NzU5ODU1MjU0MDIxNDkyNjc1NjY4NzU4MzI1ODIzMTM0MDc4ODA3OTA1NDQxNTMwMTIiLCAiMyI6ICIxMjk4NTUzMTUwMDgyNTk2NzkxODc5MTg1MzQxNjEwNzM2MTI5MTU2MTA0NTcyNzEzOTk1NTMyNTM3NjMzNzY1MzMwOTM0NDYzMjc5MzI1MDIwODAzMzkzNTAyMjA1OTk5MTYxNzc2Nzk4NDQ1NTM4MzI2MTEzMTk1Nzc1NzIxMTk0NzQxMDQ2MjE5NTY3MzI1ODc5MTQ0MzkwNTI2OTAzOTU0ODgyNzczMjU3MTY2Nzc2NzczNzU0NzkyMzYzODU1MjgyODk4NTUwNzExNDI3OTQ1NzAxNDk1OTI5NTAyMTgwNDYzOTQxOTU3ODc2MDU2ODE3ODIwMjU3ODYzMjMwOTQ3NTQ3OTUxMTUwNzk2MjU0Mzg0MzQyMDEwODM3Njc0NzQ1NTE2NzE4Nzg5MDEyNzU0MzIzMTMwMDUyODQ0MTYyNjIzNTExMjYyNzE4NDU4ODUxNDU2NjczNzYyMjkwNTY0OTczMzk3NjA0NDUxNDMyMTQwMzg0MTk1NDY5NTk5ODQ4NjczMjU3Mjk3NjY1Mzk0NTQ3NzQ0MjgxMDM1MjA2NTM0MTU5MzI4NjEzOTgwMTUyNjUzMjg1ODQ1ODQ4ODg0MzA0NTE5NzY3NTIxMjY4NTI5NDM5MDQyNDczMjEzNjAxNTI5NDIxMzQwNjA1Njg2Mzk1MjEwMzg5NTQ3MzA4MjMyOTM0ODQ4MjQ3OTA0OTU3MzA3NDQ3MzQ4MDU1MTY1MzU3OTU4MTEwMDQ0NTgxNDE4NTYxOTk1MTAwNDM4Njc5NDMzODU0MDI4MjA0NjEyMTkyMDgxNDk2NzgzNzgzMTQzODg4MjQxODkyOTM3Mzc0MDIyNjk3MjgzNjU0OTQ4MTY5Nzk5Mzc2ODgxNDUzNDQ3MTc3NjQ2MjQyNDU2NDY0NTAzNTUxMzI1NjIzODM0MjkwNzc2ODc4OTc5OTA2MjUzODI4NDM1ODM0IiwgIkRFTFRBIjogIjI1NDc4NTY3MjQ2MDcwMzI0NDUzNjU0ODkwNTcwNDA1NzI5MzcxMDA2MDU3MzExODU4NTM5MjM3OTI0NTAxNzczMzY1MTIwNDEzNTUwNTIxMzQ2ODU2MzcxNjkyMTQ1MzE1Nzc0NjQ0NTI2NjIxODY4MTQyMDQ3MzUwNjM5NzE1MjA4MTk3NTIxNjgwNzg2OTU1NjM4ODA2MTMxMzgzNTEyNjQzNDAyOTI4OTMzMzM0MTY0MDAyNTYzODg5NTk3NTc5NzEwODM3Njg0NDYwNTU4MTkxNDMwMjc3NDAxNTYwNDg5NjI2ODY2NTE3ODYxNzcwNjc5OTE2ODc5MDYyNjU2NDg2Mzg4NTg0MTkyMjQ2MzU1NDQ0NTcwNDA4MDI1NDIxNzQzODk2NjEyNDE3NzUzNTU1NjM4NjI3MDIxMzk0NTgzOTE5NjAxNTg0ODMyODYyNTk2MTEzMDk4NzI1ODU3NDAyNzQ2NTU5NDcwMDU0NDA5Mzg2NTQxMTg1NDAzMzM5MDMxNDY4MzQxMDcwOTI2NTgxOTY3MTIzNzc1MTg4ODU3Mjc1Mjk3OTI3MzI2MjkxMjU3NTczOTAyNjcwNDIwNDcwNTUxOTk2OTAxODUzNDU5NjE3OTAxNjU1NjU4MTk4NDc1NzE5NDMzMTk3NzU4NDQyNDUzNjU2NjQ5OTg0MDAyMjk4MTI3OTA3NzM3ODUxMDk3Mjg3MDk1MjcwOTkxNzMxMDU2NDk0ODE4NjI0Mzc2ODgyODUyNzI4NzQ4Nzc0MDc5NjkyMDcxOTIwNzcxNzA0NjI2MjI5NDY1MDUyOTI1ODkzNTY4NjkxMTMyNzU2NzAxOTQ2OTcyMzkxNjQxMTcyMjU5NzU3OTQzNDQzMjE5ODE2NzU3NDM4MDQzMDA1MDY0Nzg5MTI4NDgzNzczMzUxODU4NTc0NjQ0MjUzMTI2MzY2Mzg5NjU5NzIiLCAiMSI6ICIyNjM0OTUwOTI3NDcwOTQ5MjQwODMyMTg3MjUyMzQzOTYzMTI4OTEzMzE0NzYyODI1MjQ2NDgzMjQzMDExNTYyODA2OTY4OTA3NjAxMzg1NDYxMTM5OTAwNDY3ODk2MjQ4MTQyMTc1NzIwNjU5MDg3NDA2Mjg3ODYwNzMwMDk2NjE4NzUyOTg2MTQxMzc4NDc4NzA1MTIyMDQ0MTI1NjM2NTAyMTA0OTk3NjQ2MzA4Mjc3Mjc4MjAzODU0NTM3NDgyNTA1MTAwOTA0NTMzOTc1OTAwOTQ1MjgzNDE3OTI3MDY5ODU1MjU4MjgzNTE2MjczODc2MTUzNDI2Nzc1NTY3NjExMTA4NzU0NDY1MDQ2MjQxMDAzOTIzMjg2NDcyNjc2MDQ3MjE3MDQ3MDg2NTc4MjU4NDM3NjkwNzAxNDc3OTI4MzU1ODIwNzQyMTEyOTE3MjY3Mzk0NTY4NjE3ODQyNTcyNzUyODg3MTIwMjUzOTY3ODk1MjA3MDY4NTk2MDAyMTY0NjY0MjczMzg0NTk5ODM2NTYxMzU4ODU2OTAwODM1MDk1MDE4MTY0MDg5MTEwMzg0NTMyNDc4MTE0NDU5NzE4MTQxNjgwMzA3MzczMDczMDU4ODQ4NDY4NTcyMTc0MDIxNzQ3NzI5ODA5Mzg2ODE0MTU5NzUyMTIwNDA0MDE1NDI3OTk4ODc3OTE1NjkzMjUzNjMxNTA3NTg0Nzc4ODkxNDg1MjI0Mzc1NjUzMDg2MzUyNDk0MTI3ODI5NDUwMDQ3NjExMTAwNzI4Njc0Mzk1NTU2MjYyMTU3ODQ0NTE1Njk4MjIzOTkzMzU4MzcyMTYwNzA5MTQ2NzM3MTI0Nzc0NDA3NDYyODQ1NTczMzI0ODQyMDgxOTYwMzE0NDM1OTU0OTk2MTIwNzQ5NjE5OTUzNzUzMzU1NTgzOTI4NTk3NDMzOTE2NzI4NjI4IiwgIjAiOiAiMjAzOTYyNTQ2NjA3OTYzMzUxNzI3OTEwNjYyNjgzMzQwMDQyMzU4OTI0MDI5NjU3Nzg4OTc3NjM3MTUxODIwNzIxNDEyODI5MzM0MzU1MDU5MjAzMDQwMzI1OTkxNDcwMzg2OTc1NTQ5NzgzMTEyNjA1NDI2NDkxNzY3MzI2NTAwMTg2NTA1ODU5NTkzODk4NDkwNjE5MTM4NTY5NDE3ODkxNzA4NzkzMjAzMzcxNDMzMjI0NjYwMjc4MDk4ODA2MDU5MTQ0NDI1NzU0MjcyNDQ3NzU2MTAzNjAwMzkyOTExODU1NjQ2MjAzMzI5NzQzMDE1Mzg0MTk4MTEyNzczNDY2NjU3NzA5OTU3OTcyMzEyNzQ4NTAyMjc1Mzg0NDI4ODc3MTcwNzcxMDY4MzU1MTk5NjQyMTYyNDA5ODQ4NzQzODgxNDE1Njk0MjI3MDk3ODcxNzE4MjMwODk2NTg5MjQ1NDk2MTU0ODEwOTQ0NjIzMzk1Nzk4NTU4NDA1MTg4NjgyNTMwMDY5NTEyMTgwMDA2NzAyODA2MTU0MDkzMjI2NTAzNjYxMTUzODIyMzIyMzgwNjU1NDE5NzEyNDM0NjQ2NTk2MDA3MzkyNzY5NDkxOTcyNzUxNTM3NjUzNjcxODEwNjI0NjUwNjQ0OTI5MzYxNTkzMTUxNzM4MzAyOTgwODY5MTIxMTM5NzA2OTM2NzMzMzk3ODU1MjQ4ODI5NzE1NzY5ODYzODIwNDEzODcxMjUwMTQ2MTk5Mjg4NDUwMDU3MzY2NDQ4MzQ1MjY0ODkzNjA2NDczOTU4MjE3NTI1NzY1NTQwOTU1OTc0NzI1ODkyMjY5NjgwMjMyMzAwMzMyMDU5MjkwMjE1MjMyODU1NjQ3OTgwNDU0OTAyNjk5OTM4MTMyNDQ1NTM4ODM2MTM0ODI4ODYxMTQ4NzE3Njc5Mjg4NDMwNDQ3Mjc3MCJ9LCAibWoiOiAiMzI0MDYxMTk3NjY3NDI1MDc4Mzg0OTc1MTE4MTg0OTMxMzk3MzY2ODk3MjE2NzE4OTY0MjM5NjAxMDc3NDE2NjcxODk3OTQ3OTM5MjEwNTM5NTg2MzEyOTkzMDk4MzE4MjcxNDU1MTg3NjkzNjUyMjc1MjE3ODMwMjA2NDc0MDQ3NTY3NzcwNDY4MjEzNjEyMzA3OTM1NTI2MTM1MzE5OTE3MDQ2Mzc0MTYyMTgwMTM1OCIsICJhbHBoYSI6ICIyMzE0MzIzNDAxMzg2NzY3MzcxNzg2MDcyMTEwODA0MzA1Nzg5ODIzMDAyMzE5NDE1MTE1MzA2MzM0ODQ2MDMyOTMzODYwMjg0MjA3ODExMzY4ODQ5MDM0ODMyMjY0NDgwNjc4MjM2NTQyNjMxMjc5MDI4MDM1OTEyMjA2NTM5NzA4MjE3MDMzNzgzNTYwMzU3MDYzNjk0Mjk2ODU2NDEyNDk4OTE3MzIwNTMxNDM1NzczMDU5NjkyMjkwNjgxMTY2NzE3MzMzNjI3MDQ5OTM2OTk5MzE4MzI5NzY2NTk0NTA2Mzk1NzUzOTM3NTQ3NjI5NzY2NjI5Njg1MTAyNjI0Mzc1MDEzOTkyMTg1NjAxMzkzMDI1MDg5NzE4NTQ4MTM2MDIyMjI3MTcyNjM5NDU5MDYxNzg5NTA2NDE3OTQ3NzQ0Mzg4NTg4MDkyODMzNDgxMTE3NTQwMDgzMTA4NTM1OTQ0MjEzNzc3NDI0NTA4ODM5MTgyOTc2NDUyMjQyMjk5MDYyMjQxMjk2MjIxMjYxMjM1Nzk2NDgyODU3OTc2OTM4MDMxNjc2NDMxNTE4MTIxMTQ4NDI2OTUxNTQyNTk3NzI1NzE4NjcwMzcwNzUxMjE4MzA2OTM2NDEyNTQwMTI5Njk3Njk4Nzg0ODU0OTgwOTA2MzU0OTU2Njc0NzI1MDk1OTI3MDIzODg5MTg4MTIwMTA1NTQxMjUxODkxNzMyNjA4NjQ1MTgwNTI5NjM1OTA5NTc0NTQwMTI5MTkzOTE0NjM0NTUwODE1OTc3NDkzNDk0MzM2NjE0ODI2NzkwMjk1OTI3MTExMjEzMjQ1MjYzMzA5Mjc2MzczNjU3ODc2ODQ5MTU1ODEyODc4MTUzNjc2NTc3MTE5MjU4NjAxODI5NDI0NjY3MTAyOTczNjExODg5Mzc5NDU0NzM2MjI0MjI3ODkxMTgwMTQ5NjI1MDk1OTE4ODU0NzM5MzIxMDk1MTM5MzM4ODE2Mzc2MDk4ODYxMDIzODEzMzMzMzc0NjQzMTY4NDk1NjM3NjY5OTk5NTkyNTQxNTk1Nzg3NzczNDY1NTU1OTMxNDg3OTU1NDQ4MDU3MTI5OTk2OTgxMzE3MjEyMjAzMiIsICJ0IjogeyIxIjogIjM3OTI5MzcwMTI2NDExNjUyMTYwOTY4ODU4NDY3NDE2NzA0NDAzNTIzOTg5NTg3OTAyNjQxNDU0MzQ4NjU0Mjk1MDM0NTMyOTQyODQwNjkyMzM2OTIzMDY5NDk4NzAzOTU3ODczNDExNDQ5MTU4NDU0NzU3Nzc2NTMxMDM2ODYxODk2MDI4OTg5MDA0MTEzNjA5Mjk5NDg1Njk3MDgxMzg3NDc1MTQ1NzcwNjY3MDI0MjA5MzQ0MjA5MDU1MzI0Njk2NTM0MDU0ODY4NTAxMjk5MTc0MDczODY5MjI5MjAyNjc4NTk0MzU0Mjg4MzQ0NTUwMjA2OTIwMjc0OTg2OTg4MTg1Mjg3NjMwOTcyNjgyMDAwMjY5MTE4NTQ4MzcxNjgwNjY5ODM1ODM3ODI0MzIzMzEzMDIwOTA3NTgxODQ1MjM0OTEzNzM3NzgwMzE2OTY4OTg4ODk1MDAxNDQzMDY2MzIwODc0NjU4NDU2MTczODA3NTc1MTQ2OTQ0OTQzNjUyMjMyODE1NjgyOTQ5MzIwMTczMjg5MDEzMzM3MTg4NjgzMjg4MTUzMTY4MzY5Mzk0NjI5NTI1MzcxMDcwOTgwNzkzNzMzNzg4MDQyNjk5OTI3NzM2ODM1ODgwMDg1ODM0MDU3OTczNTYyOTY4MTEzNTMxMDQwNjI5MzU1NTE4MTg2MTg0MDU4ODIxOTA5ODIyMzkzMTMwNDg0NjE5NzA4ODU1MjY4NDExODc2NjEyNTQ3NzI3NzA0NjkxNTUxMTg4ODY4NDc0NzE1NzY5NzEyODMxNTI1NTAyNjM5OTQxNDgwMTcyMjQyIiwgIjMiOiAiOTY1MjY5ODczOTE1MTQ5MzM4MTA0NTk4MzE3MjM1NTQyNjU3MDkzNjk0OTY4NzM1OTIxNjI3NjcxMDQ3MzA0MzI1ODczMjM0MDk3MjUwOTg0NjA2NTE3NDc3NDQyNDA2MTE4OTE0MTQ1MTc0NjIxNzc5MDQ0Mzk4NzQ1MzczODM5Mzc4OTE0NjgwODc5NzcwNjg1MzUxMzYwOTE0MDcyOTM5OTUxOTk2ODI5MzE4NjIyNDQ5NzIxOTE2NzY0MTU5Nzg1NjAxNTg4MDg2NDE4MDkxOTkxMDc2MzUzODY0MjI2OTgyNDU5Mzg3NTgwNjE3Njg3ODEyNzcyOTIwMTU4MjA5NTg1MzIwMDU3NDc0NDQ1NjE1MjcwMTY5ODE1MjQ3ODkxOTUwMTU1ODgwNjE2MTA4NjYzNDQ3MDA5ODA5Nzg5OTc5MDg5MzcyNDU1NDIwMjEzNDU1Mzc2NzA1MTM0MTgxNzgxNjc2ODQ5MDYwODc0MjgxMTA4MzQ1NTAwNDIwMDQ4MzI0MjE4MTM0NzE1MjU1MDMyNzQ5MTc1OTA5NDc3NzMwODUzMTEwMzcxMzE0MjQxODUwNjAxOTEzMDc5NjE2MDQ1NzA3MjMzODY1OTQ1NTAwNTc2ODgxODgyNzk4MzM0OTg3NTA1MTI3ODgzODIwMDMzNTU2MDY1ODQyNzY2Njg1MjM5OTEwMDg5NDYzMjk1OTk2NTQ4Mzc1MTcyNzIxOTM0ODcxMjUzNjYzMTk2MjQ5NjYwODc3MDY3ODE2MDkxNjQ1MzYxNzg1NjQwMTUyNjI5NTkzNzU5MDkyNzA3Mjg1NDU1MTMiLCAiMCI6ICIzMTE4OTk3NjU2MDA2MzU1NTUzMTAyMTQ4MjMxODU4NTg1MTgwMzA0MDUzNDQ4NjM5MTQ5NjY5MjQzNDMwNDY5MzA0NjA3OTk4MDk2NzQ4MTE4NTU3MDg1NjAzMTk2ODU0NDAwNzM1Mjg3OTkzNDQ3NzcxMTQ4MzE1MDY3NjI2Mzk5ODk5MjMwODQ4Njg5Nzg1MDE1MTk1MjU1MjQzOTUyOTcyNTE4NTUxNDMyNDkxODg0NDA2MDI0Mjc1NTMxNjM5ODgwNTExODEzMDExMzE3MDA1NTQ1NzAwMzQ0NjA3NjI5OTAwODA0ODA5NzE1ODU2OTI5MDk3NzUyMjAyMjM5NzE5NjIxNjE5MDQwMzAzNDQxMTE2NzIxNDM1MDU4OTEzMzA5NjEzNDQ2MTUyNDk4NDQwNjMxMzM3NDExOTQwNjYyMzU0NTI4NDc1OTU5Mzk1NDE1NjA0MTA5OTgyODE4ODY3OTIwNDcxMjIzMTEwMTUwNDEwNzAyMTMzNzg0MzA5NTk2NTk5NzgwODk0Nzk1MzczNzI0NzE0ODEwNTQ5NjczOTE2OTUxODkyODMzNDcyNTU5NTQzNzcyMTEzNjAwODg2NDA0MDM3Nzk2Mjg3Njg3MzkxNjIwMzYzNjYzNzA2NjA4MDU4OTI2NjUwMTU4MTM2MzI4MjMyMDEzNjc3NjMxNzM2MzIzMzQ0MjU5NzgxNTM1NTQwNjE2NTQ1Mjc5MzI5NTIzNjU2MjcwNDg2ODc3ODY3OTM4MDM5MjcyNTM0OTU2NzM1Njc2MzY4MzcxMTcxNzExNTQxOTE0NDY5NDE4MjMzNjQzMCIsICJERUxUQSI6ICI0NzM2ODQ2NzE1MjIxNjE1NTYwNDgzNjkyMDMxMjA2Mjg0MjQ2MzM1NTIzNDk3MDM2MzAyNTExOTE2MDI4Mjg3ODY0MTYzNTMzNjQzMzkyMDU0MTQ0MDA5NjA1Mzg1NDEzOTM0NjI2MTA3ODU3OTg0NjkyNTM2NzkyMDUyODYyNDc3NjYwOTMyMjYzNTI2NTk1MjU3Mzk2NzAwOTE3Nzg4NTg2NzI0NzkxMjgxMDEzOTY2MTE3MjE4OTg2NDEyMDU1ODgzNDM0NzIxMTg1NTgyMTQzODAyMjI1MTI1NjUxNTYwNjg2OTYzMjQ5MzI1MzQ0NDkxNDA5OTk3NTk0NDA0MjgxNjc4Njk4NDMyOTUxMTcxMjkwODMzMjg4MzgzMzkzMDc2Njc5MTcxMzQyOTkyMDQ1NjI3NjM1MzQzMTE3MjYwMTE0MzQxMjE3OTAwODUxNzkyMDgyOTE4NDg1NjM2MDc2NjE2Nzg2Mzc3Mzc5NjcxMTM1MDgwOTM5NTQwMzA2NzYyMzk1Njc3MTY1MzY3NTEwOTQ4Mzc1NjU1NDYxNjgxMTE5Mzk5ODE2MDMyOTQwOTMxNDgwOTQ2NTY4ODA4Nzg0ODY2NzgxNTczODIzMDg1NDE0NzI3NjI4NjcxNzg4Mzk2MzQxMjM1NzEwNDUxNDMwNzkzMTA2OTU3Nzg2NjM4MTQzMzkyODA0ODQxMjY1MDIyOTM2NzUzNjM1MTM2NjY1ODg4MTY0MTI2Nzk3MDM0MTEzOTUwNTI0MDg2ODk1MTY5ODA0MjMyOTMxOTcyOTYxNjYyMzI4MzY0NDYyMTczNzc4NDI1IiwgIjIiOiAiMTAwMjE3MzMzNTY0MDYxMjI1NjY5MTQ3MTY1NjYzOTk3NTg2NzIwMzM2ODE5NTQxMjYyODQyMjYyMzIzOTM2MzMzMTg0MDg1NjA3MzIzOTEzNDM2NDgxMDQ3ODIxNDU1NDEwOTUxMjUzNzAxMzA1MjcxODI4NzgyNDM0Nzg5MDQxMTgxMDYyMTkxMDM2NjkyMDU3MDE4NjY4MDYyMzI2MTc1MDU2NjA2ODk0NzkwMDI0NDg1MzIzNjYyMjEwMTI3NTY4NjM0MzI0OTYyMzYwNzU5NjIzNDM5NDU4NTAxMzEwMTc1ODU0MDc3MjMwNTc5OTUyOTc2MTYxMDI2ODg0NzMzNzQwODcyMzUzOTM3MTM4OTMwNjk3NDI4ODk1NzAxMDg4MTQyMDQ3NzQ2Mzk0NDk3Mjg2MDkyNjk3NTc0NTkxNDg5Mjg4MDcxMzIxODU5MTQ1MjUxMzQwMzQ4MjAwNjY3NzExNzgyNzAwOTE1NjQxMTY4MTMyODYxODEyNzE4NDczMjUzMjY4MDY3NjkyMjkzOTM5NjU0MzIxNTQ0MzM1MzM0NzA2NzI2Mjg3NjI1NTA4NjQyMTExMTc0MTMyMDI1NTYwOTI1MjY4MzY0NzUxMTg1NTc0MDA2OTY2MDcwMzkxNTk2Nzg3MzE5NDQ4Mzc2NjQ0MDg4NTk4Mjg4Nzc2MjkzODE1MzYxNTk1NzkyMDI0NDEyNDk1NDY3MzQxMzAwNjc5OTQ2MzEzNzAxNzQzNjc2NzExNjIxNTc0MTA3NTc3NTcwMTQ1NjMwMzMzMjI3MTE4MDk1MDMzNTM4MTIwNzQ4NzA2MTM0In0sICJwcmVkaWNhdGUiOiB7ImF0dHJfbmFtZSI6ICJhZ2UiLCAicF90eXBlIjogIkdFIiwgInZhbHVlIjogMTh9fV19LCAibm9uX3Jldm9jX3Byb29mIjogbnVsbH1dLCAiYWdncmVnYXRlZF9wcm9vZiI6IHsiY19oYXNoIjogIjcwMTUyMjAxMzI1OTQ4MDMyNzUwMjU2NzQ2NDA3ODkyNzM0NTQ4Nzk0NjM1MTE5NzU2OTExODI2MTU5Nzg2NjY0ODc2NzY4NDQwMjAxIiwgImNfbGlzdCI6IFtbMiwgMTg4LCA0OSwgNDAsIDczLCA0OSwgMTE1LCAxMiwgODEsIDE2OSwgMjU1LCA5LCAxNDYsIDksIDMsIDg4LCAxODUsIDE2MiwgODQsIDE3MywgMTA0LCAxOTIsIDIzNywgMTgxLCAyNDYsIDE5NiwgNTMsIDExMCwgNDUsIDE4MCwgODEsIDEyNywgMjEzLCA1MCwgMTI1LCAyNywgMTMzLCA3MywgMTQ4LCAyMzQsIDI2LCAyMjIsIDYsIDUwLCAxODEsIDI2LCAxNjUsIDEyNywgMTkzLCA5LCAxNTksIDE2OCwgOTcsIDk2LCAxNDQsIDExOCwgMTQwLCAyMDMsIDE3OSwgMjEyLCAxNCwgMTcsIDEyMywgMTcyLCAyMzgsIDIxMywgMTgwLCAxODksIDE4NywgMTgsIDYsIDE2LCA4LCA0NywgNzEsIDM1LCA0NiwgMTM4LCAyMTcsIDE1MSwgNjIsIDE3MiwgMjYsIDIwMiwgMTI3LCA5LCA4NCwgMTI2LCAyMTIsIDI0MiwgMjU0LCA0MywgMTA2LCA3LCAxLCAxMzMsIDIwNiwgMTIxLCAyNTAsIDUxLCAxNzMsIDIwNSwgMjQxLCAxNjEsIDE3MiwgMTQyLCAyNDgsIDE0NSwgNjAsIDI1MywgMTY4LCAxNTksIDE5NiwgMTkxLCAzMiwgMTEwLCA3OCwgMTk1LCAyMzUsIDE4OCwgMCwgMTgyLCA1MywgMTMxLCAxMTYsIDIxMiwgMjEzLCAyNDQsIDEsIDY4LCAxMTgsIDgsIDM3LCAyMzUsIDI1LCAyMywgMjgsIDE3OCwgMjMwLCAxMTMsIDk0LCAxMDEsIDEyMCwgMjAyLCAzMywgMTcyLCAyNDYsIDE0NiwgMTI5LCAyMDYsIDIwNiwgMTg2LCAxMCwgMTI1LCAxMDksIDExNSwgMywgMTgzLCAxNzgsIDIwMCwgMjAxLCA1NiwgMTY4LCAyNTEsIDMxLCAzMSwgMjQ2LCAxOTcsIDEyNiwgMSwgMjAxLCAxOTQsIDIzNCwgNTAsIDI1LCAyMjYsIDE1MCwgMTAyLCA5LCA0MywgNDMsIDI0MiwgNTIsIDIzMSwgMTIwLCAyNTMsIDE5MSwgNTksIDU4LCAxMjcsIDI0NCwgMTM4LCAxOTMsIDIxNCwgMjYsIDIyMCwgMjE4LCAyMTksIDEzMywgMTgzLCAyMTUsIDExNCwgMjEzLCAxOTMsIDIwNywgMzUsIDUzLCAyMTIsIDc0LCAxNzEsIDY1LCAxMDUsIDEyNCwgMTM4LCA3MiwgMjQsIDEwMywgMTYyLCAxNzgsIDE4MCwgMTE0LCAxODQsIDkyLCAyNTAsIDk1LCAxMDEsIDEyMCwgNDQsIDE1MSwgMjA1LCA5MCwgNywgMTMwLCAxNTksIDk5LCAxMCwgMTY1LCAzMiwgMTYzLCAxMjksIDE1NywgNjcsIDcsIDE0MSwgNTMsIDE3LCAxNzcsIDM1LCA0OCwgMjI3LCAxMjcsIDY1LCAzMiwgMjI1LCAxNzYsIDI2LCA2NV0sIFsyNDcsIDE4LCAxMjQsIDMzLCAzLCAxODYsIDExNSwgMjE5LCAxMjAsIDIxOSwgMTI1LCAyMzcsIDE2MSwgMTk5LCAxODQsIDE5OSwgMzEsIDE1NiwgMTEwLCAxNSwgMTI4LCA5LCAxNDksIDE1MCwgMTcxLCA3OSwgMTcxLCAyNDcsIDE0LCA5OSwgMjQ4LCAxNTAsIDM0LCAyNDcsIDIxNSwgMzEsIDY2LCAyMDMsIDE5OCwgMjE3LCAxODIsIDIyNiwgMjQ3LCA5MiwgMTU4LCA0LCAzMSwgMTI5LCA1MCwgMzUsIDIyNCwgMjI4LCAyMzAsIDE4MiwgOTUsIDIwMSwgMjMwLCA3NiwgMTk5LCAyNDIsIDEwMywgMTU5LCAxOTgsIDM5LCAxMTIsIDEwMywgMTgyLCAyMjgsIDEwMSwgMTY2LCAyNDMsIDI2LCAxMzUsIDM0LCAxMDUsIDI1NCwgMjQ4LCAyMTMsIDM0LCAyNSwgMTY2LCAyNDgsIDMwLCAxNDcsIDc1LCAxMjksIDI0NCwgMTI0LCA4NCwgMzUsIDE5MSwgMjExLCA0MCwgMTU4LCA3MCwgMTcyLCAzNiwgNzIsIDIxLCAxMjEsIDE0NSwgMTUwLCAxMjQsIDE1LCA0MCwgMTAzLCAyMDAsIDI1LCA5LCAzOSwgMjAsIDExLCAxLCA1OCwgMTgxLCAxNzQsIDE2MiwgMTgzLCA1NCwgMjA0LCA5NywgNjAsIDkxLCAxNjEsIDEwNSwgMTI3LCA5NCwgMTQ5LCA0MiwgODMsIDQ2LCAxODQsIDExNiwgMCwgNzIsIDI2LCAzLCAyNDUsIDk3LCAyMzIsIDE4OCwgOTcsIDY0LCAxNDIsIDM4LCA3NywgMTgsIDExNCwgMTUxLCA0MiwgMTA1LCAxNjMsIDM3LCAxNjQsIDksIDMxLCA5LCAyOSwgMjcsIDE4MSwgMTY5LCAxLCAyMjMsIDM5LCAxODUsIDI0MiwgMTQ1LCAxOTEsIDEwMiwgMjE3LCA1MCwgMjExLCAxNDYsIDYsIDY2LCAxNzYsIDE2NCwgNTcsIDE0NywgMTIsIDE2NiwgNjAsIDIxLCA2NCwgMjExLCA4NSwgOCwgMzQsIDE3NSwgMzgsIDI0NCwgODksIDEyNSwgNzgsIDMyLCAxOTMsIDYyLCAxMTQsIDI0NCwgNTksIDEzNywgMzksIDEwOCwgNiwgMTUxLCAyMzUsIDExMywgMTEyLCA3NiwgMTg5LCAxMCwgMjQxLCAxNDMsIDI3LCAxNDksIDI2LCAyMTIsIDE4NSwgMjMzLCA0NywgMjA5LCAyMjYsIDgsIDE5NywgMTMxLCA4LCAyNDQsIDg4LCAxMjgsIDY5LCAxODIsIDEwOCwgNDksIDQ5LCAyMTIsIDI0NSwgMTgxLCAxMDMsIDI4LCAxMDcsIDIyNywgMTcwLCAxOTIsIDg1LCAyMzEsIDQzLCAxMzcsIDE0MSwgMTEsIDE3OSwgMjAyLCA2NywgMTMzLCA3MCwgODcsIDE3NF0sIFsxLCA0NCwgMTE3LCA5NSwgMTA1LCAxMDQsIDEwMCwgNjYsIDE3OCwgOTAsIDU4LCAxLCA3MiwgMTk1LCAxNTQsIDE5LCAxMCwgNDQsIDE0MSwgMTMzLCAzMCwgMjA4LCA0MSwgMjA4LCAxMjgsIDI0MCwgOTQsIDcxLCA3MSwgMTcxLCAxOTksIDE0MCwgNTcsIDE1NCwgNDQsIDI1MSwgMTkwLCA0MCwgMTE2LCAyMDksIDIwOCwgMjUwLCAxMiwgMTE2LCA3MSwgMjE4LCA3NSwgMTg3LCAxMjcsIDkyLCAyMDYsIDE4MywgMTg0LCAyMTMsIDIxNSwgMjEzLCA4NywgMTg1LCAxOTQsIDExOCwgNTYsIDE2NiwgNDQsIDE1MywgMjI5LCAyMjEsIDI1MCwgMjQ0LCAxNiwgNDQsIDkxLCAxNjksIDQ2LCA2MiwgNDcsIDIyLCA5MywgOTMsIDExNCwgMzYsIDgxLCA0OSwgMTY1LCAyMzcsIDE4NywgNDMsIDQ4LCAxOTYsIDM5LCAxNzEsIDQ5LCAyMjIsIDE2NCwgMjMsIDE0NywgMjIyLCAxOTAsIDgyLCAxNDAsIDIxNCwgMTEwLCAyMjksIDIyNCwgMTcsIDE2MSwgMTQ2LCAyNTIsIDExLCA0NCwgMTE5LCAyNSwgOTEsIDU0LCA5MiwgMTEzLCAxODcsIDIwNiwgNjIsIDE5MiwgMjA0LCAxNTcsIDEzNywgNjAsIDExLCAzMywgMjIxLCAxOTcsIDE3MSwgMTk1LCAxNSwgMTEsIDI1LCAxMDUsIDIyLCAxMDIsIDM2LCA1OSwgMjA3LCA2MSwgMTAsIDE1MywgODQsIDU1LCA0OCwgMTMzLCAxMSwgOTIsIDIwNywgMTkzLCAyMywgMTQ3LCA5LCA0MywgMjQzLCAxODAsIDI0NSwgMTk3LCA2OCwgMTQsIDIxNSwgMTcsIDk4LCA2MiwgMjMxLCAyMjEsIDExMiwgNywgMTc1LCAxNTksIDEyMywgMjIzLCAxNjUsIDEwNCwgMjMsIDU1LCAyNDksIDAsIDg4LCAxNzIsIDIxNCwgMjA0LCAxNjIsIDI1LCAyMzUsIDE2MiwgMjMyLCA1OCwgMzIsIDI0MywgMzUsIDYsIDE5NywgMTA4LCAyMTcsIDExOSwgMjA3LCAxMTUsIDIzMiwgNDUsIDIyMiwgMTc0LCA2OCwgOTYsIDEyMiwgMjI5LCAyNDUsIDI1MywgMjMsIDkyLCAxMjksIDEzLCAwLCAyNDYsIDI1NCwgNDUsIDIyLCAyMjMsIDIyNiwgNzIsIDExOSwgMjUsIDExLCAxNywgMzUsIDE4NiwgMjIxLCA4NSwgMzcsIDUzLCA1NSwgMjIzLCAyMDksIDE4MywgMTA2LCAyMTYsIDE0OSwgOTIsIDEyMCwgMTM2LCAxMzEsIDIzMSwgMjI1LCA0MSwgMTY2LCAyNTUsIDE4NCwgMTQyLCAyMjMsIDQwLCAyMjQsIDQwLCA0NSwgMjUwLCAxMjMsIDc3LCA3NCwgMjEwXSwgWzMsIDI1LCAyMjMsIDIwMywgMTcyLCAxOTksIDE1LCAxNTQsIDE3MiwgNTAsIDE0MiwgMjEzLCA2MiwgNTAsIDQ5LCAxMjgsIDM1LCAxNjYsIDQ1LCAxMDUsIDE5NSwgMjAyLCA0OSwgNzEsIDE2MCwgMTQ5LCAyMjAsIDEwMywgMTU3LCA3NywgMjIzLCAyNiwgMjU0LCAxMDcsIDE3NCwgMTQyLCAxNzAsIDEzMywgOTcsIDE0NiwgMTE4LCAxOSwgMTMwLCA1LCAxNDEsIDExMSwgMzcsIDk0LCAxMDksIDY3LCAwLCAyMywgMTU2LCAyNCwgMjQsIDI2LCAyMjAsIDEzMSwgMjI0LCA1OSwgNjgsIDQsIDQxLCAyMDgsIDEzMSwgMjM5LCAxMjIsIDIyNCwgMjM3LCAyNTAsIDI1NCwgMjM1LCAxODcsIDQ0LCAyMTgsIDIxMywgMTg5LCAyMTUsIDExMCwgMTgzLCA2NywgNDYsIDIyMSwgMjI3LCAxNzEsIDEyMCwgMjA5LCA4MywgMzAsIDQwLCA1OCwgMTkxLCAxNzgsIDE5OCwgNjQsIDExOSwgMTYzLCAyMTcsIDIwNywgMSwgMTkxLCAxMTUsIDEzNiwgOTksIDI1MiwgNDEsIDk1LCAyMjgsIDEwNywgNjMsIDYwLCAyMzgsIDgyLCAyNDUsIDE1NCwgMTEyLCAyMjQsIDE3MywgNTcsIDIzNCwgMTk4LCA4MiwgNzAsIDQsIDY1LCAzNCwgMTE5LCAxMDgsIDIwMiwgMTk0LCAyMiwgMTIyLCAyMDMsIDI0NCwgMjIzLCAxMjEsIDIyOSwgOTcsIDIwNSwgMTgsIDE4OCwgMTQsIDIxNiwgMjMzLCAyMTgsIDE3OCwgOTQsIDc1LCAxNTEsIDExNCwgMjM5LCA0NywgMTY5LCA4MCwgMTc5LCAxMzAsIDc1LCAyNDUsIDg4LCAxODYsIDE4NywgNzUsIDc2LCAxMzUsIDE4NiwgMTExLCA3NiwgMTA5LCA0NSwgMTI4LCAyMDAsIDE3NywgMzMsIDEyNSwgNTEsIDI1MCwgMTQ0LCAyMTUsIDc2LCAxMzUsIDE3MywgMTQwLCAxMCwgNjcsIDE3NywgNTcsIDE0NiwgMTQyLCA5OCwgOCwgMTM0LCA5NywgNjQsIDY1LCAxNzYsIDkxLCAxMDYsIDE3LCA3NCwgODEsIDE4NCwgMTU4LCAyMTQsIDQwLCAxMDcsIDI1MCwgMywgMjQsIDIwNywgMTk2LCAzMywgMTcxLCAyMjQsIDI0MCwgMjQ4LCA3OCwgMTc5LCA3NywgNzAsIDkzLCAyMzgsIDIzMywgNDcsIDEyMSwgMTI2LCA4MywgMjMwLCAxNzQsIDEyNSwgMjQxLCAyMjIsIDEyNCwgNDMsIDk0LCA0MiwgMzMsIDE1NCwgMjAwLCAyMTcsIDE2MiwgMTk2LCAxMzEsIDEzNiwgMTg0LCAxNDgsIDY5LCAyOCwgMTU1LCAyMzksIDEzMCwgNDcsIDIwNSwgMjA5LCA0LCAxMzMsIDQxLCA4Nl0sIFsyLCAyNTIsIDE2NCwgMjYsIDExOSwgMTkwLCA0NCwgNzAsIDIyOCwgMTYsIDE5NywgOTgsIDI0NywgMjE4LCAxNzksIDIxNSwgMTQ3LCAyMjQsIDIxNCwgNTUsIDM5LCAxNjYsIDY0LCAyNDIsIDExLCAxMzYsIDUyLCAyNDMsIDgxLCAyMCwgOTMsIDIyNywgMjM1LCA5OSwgODAsIDg5LCAyMzQsIDEyNiwgMTE1LCAyNDAsIDExNywgMjM5LCAxNzYsIDI2LCAxNjksIDIwNCwgMzQsIDExNSwgMjE5LCAxNjEsIDExOSwgMTgsIDE4NSwgOTgsIDI1MywgMjIyLCAxNDYsIDkzLCAxNzksIDYyLCAxNTksIDUxLCA4MSwgMTkzLCAzMSwgMzEsIDE4NiwgNjksIDM0LCAxNjQsIDE3OSwgMjI5LCAxNDUsIDE1LCAxMTcsIDUwLCA2MCwgMTY0LCA2NCwgNjQsIDEwMCwgNTgsIDIwOSwgMTAsIDIzMCwgNzQsIDkwLCAyNTQsIDE5OCwgMTAxLCAxMSwgNzcsIDEzOSwgMTkxLCA0OSwgMjE2LCAxMjgsIDIxMSwgMCwgMTcyLCAxMDMsIDEyOCwgMCwgMTg1LCAxMTQsIDg0LCA2MSwgNDcsIDM4LCA0OSwgNzksIDI0OCwgMTI3LCAxODksIDEyOCwgMjE4LCAxMjAsIDI0NywgMzUsIDE2NiwgNDcsIDEyMSwgMjUxLCAxNjMsIDk2LCA5NiwgMTk2LCAxMTcsIDIyOSwgMTcyLCAxMDEsIDIyNywgMTUxLCAxOTUsIDEyNiwgOSwgODEsIDEwNiwgMTMsIDIwLCA4NSwgMTI1LCA0NiwgNjgsIDQwLCAxNTAsIDU3LCAxMzksIDU0LCAxMTgsIDE4NCwgMTUzLCA2MywgMzgsIDIzOCwgMjA0LCA1MiwgOTUsIDE2NSwgMzQsIDUzLCA4NiwgMTI2LCAxNjIsIDEzMywgMjI0LCA2NSwgMTA0LCA2LCAxODcsIDE3NCwgMjQ3LCAxNzksIDM2LCAxMTYsIDIxNywgMTkxLCA5NCwgNTgsIDM5LCA3LCAxNzcsIDEyNywgMTA0LCA5NCwgMTIxLCAyMjcsIDM0LCAyMTEsIDE5NiwgNDIsIDY4LCA0MSwgNjksIDgwLCAxNSwgMjIxLCAzLCA5MSwgNjcsIDM5LCA5MCwgMTUsIDIzNiwgNTMsIDM3LCAxNDgsIDEzMiwgODYsIDEyNiwgMjM4LCA1NiwgMTU0LCAyMzgsIDU4LCAyMDMsIDIwOCwgMjAzLCAyMjksIDQ5LCAyMDIsIDIyLCA0OCwgMjEzLCAxMywgMjAwLCAxMzAsIDE0MywgMTAxLCA3NSwgMTU1LCA4NiwgODcsIDE2LCAxMDMsIDExMywgMTI3LCAyNTMsIDE0NSwgMTYxLCAyNDEsIDIzMSwgMTY0LCAyNTUsIDIxLCA5NSwgMjgsIDIwOCwgMjA2LCAyNDUsIDc3LCAyMTMsIDE3MSwgMTA3LCAyMzMsIDIwLCAyMzNdLCBbMzcsIDEzMywgMjMwLCAyMTksIDEwOSwgMTA1LCAxMjksIDEzMSwgMjUzLCAxODEsIDE5NiwgMTY3LCAxMzEsIDE2NCwgMjYsIDYwLCAxNTMsIDE5NiwgMjcsIDIyNSwgMzUsIDE1NiwgMTM5LCAyMTYsIDk0LCAyNDQsIDE0LCAxMjksIDI5LCA1NSwgMjUxLCAxNDgsIDk1LCAxODgsIDkwLCAyNDMsIDIyMywgMjEsIDE5MSwgMTM3LCAxNDYsIDkzLCAxMDQsIDkxLCAxNTYsIDIzMywgMjM3LCAxNjcsIDEyMCwgMjE4LCAxMzQsIDIwOSwgMTAyLCA5LCAxNDksIDE0NSwgOTksIDY5LCAxNjgsIDIyOCwgMTU0LCA4MiwgMTQ1LCAxNTUsIDE5NCwgMjAsIDU1LCAyNSwgMTk5LCAyNTUsIDEzNSwgOTYsIDE1MCwgMTI3LCA0MCwgMTAzLCA2NCwgNTMsIDMwLCAxNTAsIDIzMSwgMTgzLCA5LCA0OCwgMjI2LCAyMzAsIDk5LCAyMTEsIDEyNSwgMTQwLCA4NiwgMjI0LCAxODMsIDE3LCAxNjQsIDIyNiwgMTAyLCAxNCwgNDAsIDE0MCwgNjMsIDE4NCwgMTc1LCAyMzMsIDEyOCwgMTA1LCA2NCwgMTA0LCA2OCwgMjMyLCAxNDMsIDEwNywgMjQsIDEwMywgMSwgMjUwLCAyMjksIDI1NCwgMTk4LCAxMiwgMTc0LCAxNjQsIDIzMCwgODMsIDE5NSwgMzIsIDc4LCA4NSwgMjcsIDQ0LCAxNzEsIDk0LCA5MCwgNjgsIDE3MiwgMjM4LCA5MSwgMTgsIDE4LCAyMjEsIDEwMSwgMjUzLCAxOTksIDEsIDY2LCAzOSwgMTc5LCAxNDYsIDE4OCwgMTU0LCAxNTYsIDIxMiwgMjgsIDEyNywgMTkyLCA3LCAxNDQsIDIwMCwgOCwgMTk1LCAxMjgsIDE1NiwgMjI3LCA0OSwgMzcsIDIzNywgOTEsIDE5MCwgMTMwLCAxNTAsIDI0NSwgMjQzLCAxNzcsIDEzMiwgMTU0LCAxMywgMTAxLCAyMzksIDI1NCwgMTc0LCA0LCAxOCwgMTkyLCAyMzQsIDIwMCwgMTg1LCAxNCwgMjMwLCAyNDIsIDE5MiwgMTk5LCAxNywgMTkwLCAyMjUsIDYwLCAxNDAsIDM2LCA3MSwgMjE2LCA5LCA5OSwgMjEzLCAxMjIsIDE2NywgMTQsIDE4NywgMTAyLCA0NywgMjYsIDE2MCwgMTgsIDIyMCwgMjM1LCAyMTgsIDUwLCAyMDAsIDEwMCwgNTMsIDEzLCAxNCwgMTA3LCA0LCAxNDEsIDY2LCAxMDksIDQ5LCAxOTIsIDIyMCwgODYsIDIxNCwgMTQ5LCAxNTAsIDMwLCAxMzQsIDk5LCAxODAsIDQ4LCAyMjksIDIyMiwgMTY1LCA0NywgMjM3LCAxNDQsIDIwLCAxODYsIDAsIDQ4LCA4NiwgMjU0LCAyMDcsIDIyMCwgMTUzLCAxNzEsIDI0MiwgMjQxLCAyNDldXX19LCAicmVxdWVzdGVkX3Byb29mIjogeyJyZXZlYWxlZF9hdHRycyI6IHsiMF9uYW1lX3V1aWQiOiB7InN1Yl9wcm9vZl9pbmRleCI6IDAsICJyYXciOiAiQWxpY2UgU21pdGgiLCAiZW5jb2RlZCI6ICI2MjgxNjgxMDIyNjkzNjY1NDc5Nzc3OTcwNTAwMDc3Mjk2ODA1ODI4Mzc4MDEyNDMwOTA3NzA0OTY4MTczNDgzNTc5NjMzMjcwNDQxMyJ9LCAiMF9kZWdyZWVfdXVpZCI6IHsic3ViX3Byb29mX2luZGV4IjogMCwgInJhdyI6ICJNYXRocyIsICJlbmNvZGVkIjogIjQ2MDI3MzIyOTIyMDQwODU0MjE3ODcyOTMyODk0ODU0ODIzNTEzMjkwNTM5MzQwMDAwMTU4MjM0Mjk0NDE0NzgxMzk4NDY2MDc3MiJ9LCAiMF9kYXRlX3V1aWQiOiB7InN1Yl9wcm9vZl9pbmRleCI6IDAsICJyYXciOiAiMjAxOC0wNS0yOCIsICJlbmNvZGVkIjogIjIzNDAyNjM3NDIzODc2MzI0MDk4MjU2NTE5MzE3Njk1NDMzMTk2ODEzMjE3Nzg1Nzk1MzE3MjIwNjgwNDE1ODEyMzQ4ODAxMDg2NTg2In19LCAic2VsZl9hdHRlc3RlZF9hdHRycyI6IHsiMF9zZWxmX2F0dGVzdGVkX3RoaW5nX3V1aWQiOiAibXkgc2VsZi1hdHRlc3RlZCB2YWx1ZSJ9LCAidW5yZXZlYWxlZF9hdHRycyI6IHt9LCAicHJlZGljYXRlcyI6IHsiMF9hZ2VfR0VfdXVpZCI6IHsic3ViX3Byb29mX2luZGV4IjogMH19fSwgImlkZW50aWZpZXJzIjogW3sic2NoZW1hX2lkIjogIlNhemdWcmVVWHR3RjRaQndBeFVQd1U6MjpkZWdyZWUgc2NoZW1hOjE1LjQzLjkzIiwgImNyZWRfZGVmX2lkIjogIlNhemdWcmVVWHR3RjRaQndBeFVQd1U6MzpDTDoyNTpkZWZhdWx0IiwgInJldl9yZWdfaWQiOiBudWxsLCAidGltZXN0YW1wIjogbnVsbH1dfQ=="
        }
      }
    ]
  }`

func TestPropose_Start(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	ID := "TEST_ID"
	credDefID := "CRED_DEF_ID"
	values := []string{"email", "ssn"}
	p1 := newPropose(ID, credDefID, values)

	data := p1.JSON()
	p2 := Creator.NewMessage(data)
	assert.DeepEqual(p1, p2)
}

func TestPropose_New(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	ID := "TEST_ID"
	credDefID := "CRED_DEF_ID"
	values := []string{"email", "ssn"}

	msg := newPropose(ID, credDefID, values)
	opl := aries.PayloadCreator.NewMsg(ID, pltype.PresentProofPropose, msg)

	json := opl.JSON()

	ipl := aries.PayloadCreator.NewFromData(json)

	if pltype.PresentProofPropose != ipl.Type() {
		t.Errorf("wrong type %v", ipl.Type())
	}

	o := ipl.MsgHdr().FieldObj().(*Propose)
	if o == nil {
		t.Error("request is nil")
	}

	assert.DeepEqual(opl, ipl)
}

func TestPresentation_ReadJSON(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	var req Presentation

	dto.FromJSONStr(presentation, &req)
	if req.ID != "96a5893d-113a-435d-8637-3f33ebebc620" {
		t.Errorf("id (%v) not match", req.ID)
	}

	data, err := base64.StdEncoding.DecodeString(req.PresentationAttaches[0].Data.Base64)
	assert.NoError(err)

	proof := make(map[string]interface{})
	dto.FromJSON(data, &proof)

	_, ok := proof["proof"]
	if !ok {
		t.Error("proof key not found in indy proof")
	}

}

func TestPresentation_ReadJSONAndBuildNew(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	var p Presentation

	dto.FromJSONStr(presentation, &p)
	if p.ID != "96a5893d-113a-435d-8637-3f33ebebc620" {
		t.Errorf("id (%v) not match", p.ID)
	}

	data, err := Proof(&p)
	assert.NoError(err)

	proof := make(map[string]interface{})
	dto.FromJSON(data, &proof)

	_, ok := proof["proof"]
	if !ok {
		t.Error("proof key not found in indy proof")
	}

	m := aries.MsgCreator.Create(didcomm.MsgInit{
		AID:    p.ID,
		Type:   pltype.PresentProofPresentation,
		Thread: p.Thread,
	})
	p2 := m.FieldObj().(*Presentation)
	p2.PresentationAttaches = NewPresentationAttach(
		pltype.LibindyPresentationID, data)

	readJSON := dto.ToJSON(p)
	buildJSON := dto.ToJSON(p2)
	if !reflect.DeepEqual(readJSON, buildJSON) {
		t.Errorf("not equal\nis\t(%v)\nwant(%v)", buildJSON, readJSON)
	}

}

func TestPresentation_MsgPingPong(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	inviteID := "invite id"
	firstMsgID := "prop id"

	prop := aries.MsgCreator.Create(didcomm.MsgInit{
		AID:    firstMsgID,
		Type:   pltype.PresentProofPropose,
		Thread: decorator.NewThread(firstMsgID, inviteID),
	}).(*ProposeImpl)

	req := aries.MsgCreator.Create(didcomm.MsgInit{
		AID:    "req id",
		Type:   pltype.PresentProofRequest,
		Thread: prop.Thread(),
	}).(*RequestImpl)

	assert.Equal(req.Thread().PID, inviteID)
	assert.Equal(req.Thread().ID, firstMsgID)

	pres := aries.MsgCreator.Create(didcomm.MsgInit{
		AID:    "pres id",
		Type:   pltype.PresentProofPresentation,
		Thread: req.Thread(),
	}).(*PresentationImpl)

	assert.Equal(pres.Thread().PID, inviteID)
	assert.Equal(pres.Thread().ID, firstMsgID)
}

func TestRequest_ReadJSONAndBuildNew(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	var req Request

	dto.FromJSONStr(request, &req)
	if req.ID != "b1220020-bfd6-408e-a51d-3cfcd2a99acb" {
		t.Errorf("id (%v) not match", req.ID)
	}

	proofReq, err := ProofReq(&req)
	assert.NoError(err)

	nonce, ok := proofReq["nonce"]
	if !ok {
		t.Error("nonce not found in proof req")
	}

	if nonce.(string) != "81216214931320823518806071291707691806" {
		t.Errorf("nonce (%v) not match", nonce)
	}

	m := aries.MsgCreator.Create(didcomm.MsgInit{
		AID:    req.ID,
		Type:   pltype.PresentProofRequest,
		Thread: req.Thread,
	})
	req2 := m.FieldObj().(*Request)

	// we use the same proof req parsed from original message
	data, err := ProofReqData(&req)
	assert.NoError(err)

	req2.RequestPresentations = NewRequestPresentation(
		"a1f23394-df26-4cd4-8afb-8068244ca7f9", data)
	// remove this because this is not correct test if it's here, normal case
	// we input and output messages, where it's important to create thread for
	// output message even the input message doesn't have it.
	req2.Thread = nil
	readJSON := dto.ToJSON(req)
	buildJSON := dto.ToJSON(req2)
	assert.Equal(readJSON, buildJSON)

	ack := aries.MsgCreator.Create(didcomm.MsgInit{
		AID:    req.ID,
		Type:   pltype.PresentProofACK,
		Thread: req2.Thread,
	})
	assert.INotNil(ack)
}

func TestNewRequest(t *testing.T) {
	assert.PushTester(t)
	defer assert.PopTester()

	var b64 = `
eyJuYW1lIjogIlByb29mIG9mIEVkdWNhdGlvbiIsICJ2ZXJzaW9uIjogIjEuMCIsICJub25jZSI6ICIzMTMyNDIzMTMzODYyMDExMDAyODk1NTk5OTUwNTE3MTY4MTY0ODgiLCAicmVxdWVzdGVkX2F0dHJpYnV0ZXMiOiB7IjBfbmFtZV91dWlkIjogeyJuYW1lIjogIm5hbWUiLCAicmVzdHJpY3Rpb25zIjogW3siaXNzdWVyX2RpZCI6ICI1TGd4NEtMUlROZ2V4RHFUN1dEQUx1In1dfSwgIjBfZGF0ZV91dWlkIjogeyJuYW1lIjogImRhdGUiLCAicmVzdHJpY3Rpb25zIjogW3siaXNzdWVyX2RpZCI6ICI1TGd4NEtMUlROZ2V4RHFUN1dEQUx1In1dfSwgIjBfZGVncmVlX3V1aWQiOiB7Im5hbWUiOiAiZGVncmVlIiwgInJlc3RyaWN0aW9ucyI6IFt7Imlzc3Vlcl9kaWQiOiAiNUxneDRLTFJUTmdleERxVDdXREFMdSJ9XX0sICIwX3NlbGZfYXR0ZXN0ZWRfdGhpbmdfdXVpZCI6IHsibmFtZSI6ICJzZWxmX2F0dGVzdGVkX3RoaW5nIn19LCAicmVxdWVzdGVkX3ByZWRpY2F0ZXMiOiB7IjBfYWdlX0dFX3V1aWQiOiB7Im5hbWUiOiAiYWdlIiwgInBfdHlwZSI6ICI+PSIsICJwX3ZhbHVlIjogMTgsICJyZXN0cmljdGlvbnMiOiBbeyJpc3N1ZXJfZGlkIjogIjVMZ3g0S0xSVE5nZXhEcVQ3V0RBTHUifV19fX0=`

	var second = `
eyJuYW1lIjogIlByb29mIG9mIEVkdWNhdGlvbiIsICJ2ZXJzaW9uIjogIjEuMCIsICJub25jZSI6ICIzODExMDI3ODk1ODM1NTQ5ODIwMTk4NTEyMzIxOTU2MDg0Nzk2NCIsICJyZXF1ZXN0ZWRfYXR0cmlidXRlcyI6IHsiMF9uYW1lX3V1aWQiOiB7Im5hbWUiOiAibmFtZSIsICJyZXN0cmljdGlvbnMiOiBbeyJpc3N1ZXJfZGlkIjogIjVMZ3g0S0xSVE5nZXhEcVQ3V0RBTHUifV19LCAiMF9kYXRlX3V1aWQiOiB7Im5hbWUiOiAiZGF0ZSIsICJyZXN0cmljdGlvbnMiOiBbeyJpc3N1ZXJfZGlkIjogIjVMZ3g0S0xSVE5nZXhEcVQ3V0RBTHUifV19LCAiMF9kZWdyZWVfdXVpZCI6IHsibmFtZSI6ICJkZWdyZWUiLCAicmVzdHJpY3Rpb25zIjogW3siaXNzdWVyX2RpZCI6ICI1TGd4NEtMUlROZ2V4RHFUN1dEQUx1In1dfSwgIjBfc2VsZl9hdHRlc3RlZF90aGluZ191dWlkIjogeyJuYW1lIjogInNlbGZfYXR0ZXN0ZWRfdGhpbmcifX0sICJyZXF1ZXN0ZWRfcHJlZGljYXRlcyI6IHsiMF9hZ2VfR0VfdXVpZCI6IHsibmFtZSI6ICJhZ2UiLCAicF90eXBlIjogIj49IiwgInBfdmFsdWUiOiAxOCwgInJlc3RyaWN0aW9ucyI6IFt7Imlzc3Vlcl9kaWQiOiAiNUxneDRLTFJUTmdleERxVDdXREFMdSJ9XX19fQ==`

	var third = `eyJuYW1lIjogIlByb29mIG9mIEVkdWNhdGlvbiIsICJ2ZXJzaW9uIjogIjEuMCIsICJub25jZSI6ICIxMDAyNzgwNjUyNDc0NjM0NTQ1NTc3NzY2MTMyMTA1MzM1NDAyMTgiLCAicmVxdWVzdGVkX2F0dHJpYnV0ZXMiOiB7IjBfbmFtZV91dWlkIjogeyJuYW1lIjogIm5hbWUiLCAicmVzdHJpY3Rpb25zIjogW3siaXNzdWVyX2RpZCI6ICI1TGd4NEtMUlROZ2V4RHFUN1dEQUx1In1dfSwgIjBfZGF0ZV91dWlkIjogeyJuYW1lIjogImRhdGUiLCAicmVzdHJpY3Rpb25zIjogW3siaXNzdWVyX2RpZCI6ICI1TGd4NEtMUlROZ2V4RHFUN1dEQUx1In1dfSwgIjBfZGVncmVlX3V1aWQiOiB7Im5hbWUiOiAiZGVncmVlIiwgInJlc3RyaWN0aW9ucyI6IFt7Imlzc3Vlcl9kaWQiOiAiNUxneDRLTFJUTmdleERxVDdXREFMdSJ9XX0sICIwX3NlbGZfYXR0ZXN0ZWRfdGhpbmdfdXVpZCI6IHsibmFtZSI6ICJzZWxmX2F0dGVzdGVkX3RoaW5nIn19LCAicmVxdWVzdGVkX3ByZWRpY2F0ZXMiOiB7IjBfYWdlX0dFX3V1aWQiOiB7Im5hbWUiOiAiYWdlIiwgInBfdHlwZSI6ICI+PSIsICJwX3ZhbHVlIjogMTgsICJyZXN0cmljdGlvbnMiOiBbeyJpc3N1ZXJfZGlkIjogIjVMZ3g0S0xSVE5nZXhEcVQ3V0RBTHUifV19fX0=`

	str, err := base64.StdEncoding.DecodeString(b64)
	assert.NoError(err)
	assert.SNotEmpty(str)

	str, err = base64.StdEncoding.DecodeString(second)
	assert.NoError(err)
	assert.SNotEmpty(str)

	str, err = base64.StdEncoding.DecodeString(third)
	assert.NoError(err)
	assert.SNotEmpty(str)

}
