import { createScopedProxy } from "@/lib/api/scoped-proxy";

const forward = createScopedProxy("/owner");

export { forward as DELETE, forward as GET, forward as PATCH, forward as POST, forward as PUT };
