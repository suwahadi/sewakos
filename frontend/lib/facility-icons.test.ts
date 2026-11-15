import { AirVent, Bath, BedDouble, Bike, Camera, Car, Check, Clock, Fan, Shirt, Table2, Tv, Wifi, type LucideIcon } from "lucide-react";
import { describe, expect, it } from "vitest";
import { facilityIcon } from "./facility-list";

describe("facilityIcon", () => {
  it.each<[string, LucideIcon]>([
    ["WiFi", Wifi],
    ["AC", AirVent],
    ["K. Mandi Dalam", Bath],
    ["Kasur Queen", BedDouble],
    ["Lemari", Shirt],
    ["Meja Belajar", Table2],
    ["TV", Tv],
    ["Akses 24 Jam", Clock],
    ["CCTV 24 jam", Camera],
    ["parkir mobil terbatas", Car],
    ["parkir motor", Bike],
    ["Kipas Angin", Fan],
  ])("maps %s", (facility, icon) => {
    expect(facilityIcon(facility)).toBe(icon);
  });

  it("does not match the ac rule inside longer words", () => {
    expect(facilityIcon("Kamar Acak")).toBe(Check);
  });
});
