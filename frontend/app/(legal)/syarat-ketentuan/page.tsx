import { LegalPage } from "../legal-page";

export const metadata = { title: "Syarat & Ketentuan" };

const PARAGRAPHS = [
  "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Praesent commodo cursus magna, vel scelerisque nisl consectetur et. Morbi leo risus, porta ac consectetur ac, vestibulum at eros. Duis mollis, est non commodo luctus, nisi erat porttitor ligula, eget lacinia odio sem nec elit.",
  "Vivamus sagittis lacus vel augue laoreet rutrum faucibus dolor auctor. Cum sociis natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Maecenas sed diam eget risus varius blandit sit amet non magna. Donec id elit non mi porta gravida at eget metus.",
  "Nullam id dolor id nibh ultricies vehicula ut id elit. Sed posuere consectetur est at lobortis. Aenean eu leo quam, pellentesque ornare sem lacinia quam venenatis vestibulum. Integer posuere erat a ante venenatis dapibus posuere velit aliquet, cras justo odio dapibus ac facilisis in.",
];

export default function SyaratKetentuanPage() {
  return <LegalPage title="Syarat & Ketentuan" updated="12 Nov 2026" paragraphs={PARAGRAPHS} />;
}
