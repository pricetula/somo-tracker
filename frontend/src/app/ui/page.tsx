"use client";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
  InputGroupText,
} from "@/components/ui/input-group";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const COUNTRIES = [
  { id: "us", name: "United States" },
  { id: "ca", name: "Canada" },
  { id: "gb", name: "United Kingdom" },
  { id: "de", name: "Germany" },
  { id: "fr", name: "France" },
];

export default function UIComponentsPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4 sm:p-6">
      <div className="w-full max-w-2xl space-y-8">
        <h1 className="text-3xl font-bold">shadcn/ui Components</h1>
        <p className="text-muted-foreground">
          Public page to verify shadcn component setup.
        </p>

        <section className="space-y-4">
          <h2 className="text-xl font-semibold">Buttons</h2>
          <div className="flex flex-wrap gap-2">
            <Button>Default</Button>
            <Button variant="outline">Outline</Button>
            <Button variant="secondary">Secondary</Button>
            <Button variant="ghost">Ghost</Button>
            <Button variant="destructive">Destructive</Button>
            <Button variant="link">Link</Button>
            <Button size="sm">Small</Button>
            <Button size="lg">Large</Button>
            <Button disabled>Disabled</Button>
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-xl font-semibold">Inputs</h2>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" placeholder="name@example.com" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="disabled-input">Disabled Input</Label>
              <Input id="disabled-input" defaultValue="Disabled" disabled />
            </div>
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-xl font-semibold">Checkbox</h2>
          <div className="space-y-2">
            <Checkbox defaultChecked>Accept terms</Checkbox>
            <Checkbox disabled>Disabled</Checkbox>
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-xl font-semibold">Textarea</h2>
          <div className="space-y-2">
            <Label htmlFor="bio">Bio</Label>
            <Textarea id="bio" placeholder="Tell us about yourself..." />
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-xl font-semibold">Label</h2>
          <div className="space-y-2">
            <Label htmlFor="username">Username</Label>
            <Input id="username" placeholder="jdoe" />
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-xl font-semibold">Input Group</h2>
          <div className="space-y-4">
            <InputGroup>
              <InputGroupText>https://example.com/</InputGroupText>
              <InputGroupInput placeholder="username" />
            </InputGroup>
            <InputGroup>
              <InputGroupInput placeholder="Amount" />
              <InputGroupAddon align="inline-end">USD</InputGroupAddon>
            </InputGroup>
            <InputGroup>
              <InputGroupInput placeholder="Search..." />
              <InputGroupAddon align="inline-end">
                <InputGroupButton variant="outline">Go</InputGroupButton>
              </InputGroupAddon>
            </InputGroup>
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-xl font-semibold">Combobox</h2>
          <Combobox>
            <ComboboxInput placeholder="Select a country" />
            <ComboboxContent>
              <ComboboxList>
                {COUNTRIES.map((country) => (
                  <ComboboxItem key={country.id} value={country.id}>
                    {country.name}
                  </ComboboxItem>
                ))}
              </ComboboxList>
            </ComboboxContent>
          </Combobox>
        </section>

        <section className="space-y-4">
          <h2 className="text-xl font-semibold">Dialog</h2>
          <Dialog>
            <DialogTrigger>Open dialog</DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Are you sure?</DialogTitle>
                <DialogDescription>
                  This action cannot be undone.
                </DialogDescription>
              </DialogHeader>
              <DialogFooter>
                <Button>Cancel</Button>
                <Button>Confirm</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </section>

        <section className="space-y-4">
          <h2 className="text-xl font-semibold">Dropdown Menu</h2>
          <DropdownMenu>
            <DropdownMenuTrigger>Actions</DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem>Profile</DropdownMenuItem>
              <DropdownMenuItem>Settings</DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem>Logout</DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </section>

        <section className="space-y-4">
          <h2 className="text-xl font-semibold">Tooltip</h2>
          <Tooltip>
            <TooltipTrigger>Hover me</TooltipTrigger>
            <TooltipContent>This is a tooltip</TooltipContent>
          </Tooltip>
        </section>
      </div>
    </div>
  );
}
