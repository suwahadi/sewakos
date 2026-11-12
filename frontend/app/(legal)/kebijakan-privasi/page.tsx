import { LegalPage } from "../legal-page";

export const metadata = { title: "Kebijakan Privasi" };

const PARAGRAPHS = [
  "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Integer posuere erat a ante venenatis dapibus posuere velit aliquet. Donec ullamcorper nulla non metus auctor fringilla. Vestibulum id ligula porta felis euismod semper, sed posuere consectetur est at lobortis.",
  "Curabitur blandit tempus porttitor. Maecenas faucibus mollis interdum. Nullam quis risus eget urna mollis ornare vel eu leo. Cras mattis consectetur purus sit amet fermentum, sed vulputate cursus magna, vel scelerisque nisl consectetur et. Aenean lacinia bibendum nulla sed consectetur.",
  "Etiam porta sem malesuada magna mollis euismod. Donec sed odio dui. Fusce dapibus, tellus ac cursus commodo, tortor mauris condimentum nibh, ut fermentum massa justo sit amet risus. Nulla vitae elit libero, a pharetra augue. Sed posuere consectetur est at lobortis.",
];

export default function KebijakanPrivasiPage() {
  return <LegalPage title="Kebijakan Privasi" updated="12 Nov 2026" paragraphs={PARAGRAPHS} />;
}
