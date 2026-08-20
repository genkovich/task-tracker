import type { Editor } from "@tiptap/react";
import type { LucideIcon } from "lucide-react";
import {
  Type,
  Heading1,
  Heading2,
  Heading3,
  List,
  ListOrdered,
  Quote,
  FileCode,
  ImageIcon,
  Minus,
} from "lucide-react";
import { isSafeUrl } from "@/shared/lib/isSafeUrl";

export interface SlashCommandItem {
  title: string;
  description: string;
  icon: LucideIcon;
  aliases: string[];
  command: (editor: Editor) => void;
}

export const SLASH_COMMANDS: SlashCommandItem[] = [
  {
    title: "Text",
    description: "Plain text paragraph",
    icon: Type,
    aliases: ["p", "paragraph"],
    command: (editor) => {
      editor.chain().focus().setParagraph().run();
    },
  },
  {
    title: "Heading 1",
    description: "Large section heading",
    icon: Heading1,
    aliases: ["h1", "heading1"],
    command: (editor) => {
      editor.chain().focus().toggleHeading({ level: 1 }).run();
    },
  },
  {
    title: "Heading 2",
    description: "Medium section heading",
    icon: Heading2,
    aliases: ["h2", "heading2"],
    command: (editor) => {
      editor.chain().focus().toggleHeading({ level: 2 }).run();
    },
  },
  {
    title: "Heading 3",
    description: "Small section heading",
    icon: Heading3,
    aliases: ["h3", "heading3"],
    command: (editor) => {
      editor.chain().focus().toggleHeading({ level: 3 }).run();
    },
  },
  {
    title: "Bullet List",
    description: "Unordered list of items",
    icon: List,
    aliases: ["ul", "unordered", "bullets"],
    command: (editor) => {
      editor.chain().focus().toggleBulletList().run();
    },
  },
  {
    title: "Ordered List",
    description: "Numbered list of items",
    icon: ListOrdered,
    aliases: ["ol", "numbered"],
    command: (editor) => {
      editor.chain().focus().toggleOrderedList().run();
    },
  },
  {
    title: "Blockquote",
    description: "Quote or callout block",
    icon: Quote,
    aliases: ["quote", "callout"],
    command: (editor) => {
      editor.chain().focus().toggleBlockquote().run();
    },
  },
  {
    title: "Code Block",
    description: "Code with syntax highlighting",
    icon: FileCode,
    aliases: ["code", "codeblock", "pre"],
    command: (editor) => {
      editor.chain().focus().toggleCodeBlock().run();
    },
  },
  {
    title: "Image",
    description: "Insert image from URL",
    icon: ImageIcon,
    aliases: ["img", "picture", "photo"],
    command: (editor) => {
      const url = window.prompt("Image URL:");
      if (url) {
        if (!isSafeUrl(url)) {
          window.alert("Invalid URL. Only http and https URLs are allowed.");
          return;
        }
        editor.chain().focus().setImage({ src: url }).run();
      }
    },
  },
  {
    title: "Horizontal Rule",
    description: "Visual divider line",
    icon: Minus,
    aliases: ["hr", "divider", "separator"],
    command: (editor) => {
      editor.chain().focus().setHorizontalRule().run();
    },
  },
];
