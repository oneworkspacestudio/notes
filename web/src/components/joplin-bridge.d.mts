export default class MdToHtml {
  constructor(options: Record<string, unknown>);
  render(
    body: string,
    theme: Record<string, unknown>,
    options: Record<string, unknown>,
  ): Promise<{ html: string }>;
}
