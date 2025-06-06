import { Dropdown, initFlowbite } from 'flowbite';

import { Store } from '@ngrx/store';

import { CommonModule } from '@angular/common';
import { AfterViewInit, Component, ElementRef, ViewChild, inject } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { ActivatedRoute, RouterModule } from '@angular/router';

import { actions } from '@app/ngrx/finance';
import { FlowbiteService } from '@app/services/flowbite.service';
import { Category } from '@app/types';

@Component({
  selector: 'app-create-category',
  templateUrl: './create-category.page.html',
  imports: [CommonModule, RouterModule, ReactiveFormsModule],
})
export class CreateCategoryPage implements AfterViewInit {
  private store = inject(Store);
  private fb = inject(FormBuilder);
  private route = inject(ActivatedRoute);
  private flowbite = inject(FlowbiteService);

  @ViewChild('dropdownSearch') dropdownSearch!: ElementRef;
  @ViewChild('dropdownSearchButton') dropdownSearchButton!: ElementRef;

  iconsDropdown: Dropdown | null = null;
  walletID = this.route.snapshot.params['walletID'];
  icons: { name: string; popularity: number; tags: string[] }[] = [];
  filteredIcons: { name: string; popularity: number; tags: string[] }[] = [];
  colorShades = [300, 400, 500, 600, 700];
  colors = [
    'red',
    'orange',
    'amber',
    'yellow',
    'lime',
    'green',
    'emerald',
    'teal',
    'cyan',
    'sky',
    'blue',
    'indigo',
    'violet',
    'purple',
    'fuchsia',
    'pink',
    'rose',
    'slate',
    'stone',
  ];

  categoryForm = this.fb.group({
    name: ['', Validators.required],
    icon: ['help', Validators.required],
    color: [this.generateRandomColor(), Validators.required],
  });

  async ngOnInit() {
    this.icons = await fetch('/icons.json')
      .then((res) => res.json())
      .then((icons) => icons.sort((a: any, b: any) => b.popularity - a.popularity))
      .then((icons) => (this.filteredIcons = icons))
      .then((icons) => {
        this.selectRandomIcon();
        return icons;
      });
  }

  ngAfterViewInit() {
    this.flowbite.load(() => initFlowbite());
    this.iconsDropdown = new Dropdown(this.dropdownSearch.nativeElement, this.dropdownSearchButton.nativeElement);
  }

  generateRandomColor(): string {
    const color = this.colors[Math.floor(Math.random() * this.colors.length)];
    const shade = this.colorShades[Math.floor(Math.random() * this.colorShades.length)];
    return `bg-${color}-${shade}`;
  }

  selectRandomIcon() {
    const randomIcon = this.filteredIcons[Math.floor(Math.random() * this.filteredIcons.length)];
    this.categoryForm.patchValue({ icon: randomIcon.name });
  }

  searchIcon(event: Event) {
    const searchValue = (event.target as HTMLInputElement).value;
    this.filteredIcons = this.icons.filter(
      (icon) =>
        icon.name.toLowerCase().includes(searchValue.toLowerCase()) ||
        icon.tags.some((tag) => tag.toLowerCase().includes(searchValue.toLowerCase())),
    );
  }

  selectIcon(icon: { name: string; popularity: number; tags: string[] }) {
    this.categoryForm.patchValue({ icon: icon.name });
    this.iconsDropdown?.hide();
    this.dropdownSearchButton.nativeElement.focus();
  }

  createCategory() {
    if (!this.categoryForm.valid) {
      return;
    }

    this.store.dispatch(
      actions.createCategory({
        walletID: this.walletID,
        category: this.categoryForm.value as Category,
      }),
    );
  }
}
