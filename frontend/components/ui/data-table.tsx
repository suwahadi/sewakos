import type { ReactNode } from "react";
import { cn } from "@/lib/cn";

export interface DataTableColumn<T> {
  key: string;
  header: string;
  render: (row: T) => ReactNode;
  align?: "right";
}

export interface DataTableProps<T> {
  caption: string;
  columns: DataTableColumn<T>[];
  rows: T[];
  rowKey: (row: T) => string | number;
}

export function DataTable<T>({ caption, columns, rows, rowKey }: DataTableProps<T>) {
  return (
    <>
      <ul aria-label={caption} className="flex flex-col divide-y divide-neutral-200 rounded-lg border border-neutral-200 lg:hidden">
        {rows.map((row) => (
          <li key={rowKey(row)} className="flex flex-col gap-1.5 p-3">
            {columns.map((column) => (
              <div key={column.key} className="flex min-w-0 items-start justify-between gap-3">
                <span className="shrink-0 text-caption text-neutral-600">{column.header}</span>
                <span className="min-w-0 break-words text-right text-body text-neutral-900">{column.render(row)}</span>
              </div>
            ))}
          </li>
        ))}
      </ul>
      <div className="hidden overflow-x-auto rounded-lg border border-neutral-200 lg:block">
        <table className="w-full text-left text-body text-neutral-900">
          <caption className="sr-only">{caption}</caption>
          <thead className="bg-neutral-50 text-caption-strong text-neutral-600">
            <tr>
              {columns.map((column) => (
                <th key={column.key} scope="col" className={cn("px-4 py-3", column.align === "right" && "text-right")}>
                  {column.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-200">
            {rows.map((row) => (
              <tr key={rowKey(row)} className="h-[52px]">
                {columns.map((column) => (
                  <td key={column.key} className={cn("px-4 py-2", column.align === "right" && "text-right tabular-nums")}>
                    {column.render(row)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
