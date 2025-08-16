# Copyright 2018-2021 Rahul De
#
# Use of this source code is governed by an MIT-style
# license that can be found in the LICENSE file or at
# https://opensource.org/licenses/MIT.

FROM scratch

ARG TARGETPLATFORM

COPY ${TARGETPLATFORM}/main .

ENTRYPOINT ["./main"]
