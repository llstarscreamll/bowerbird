import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FileUploadComponent } from './file-upload.component';

describe('FileUploadComponent', () => {
  let fixture: ComponentFixture<FileUploadComponent>;
  let component: FileUploadComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [FileUploadComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(FileUploadComponent);
    component = fixture.componentInstance;
    fixture.componentRef.setInput('validateFile', (file: File) => file.name.endsWith('.xml'));
    fixture.detectChanges();
  });

  it('should reject files that fail validation', () => {
    const emitted: File[][] = [];
    component.filesSelected.subscribe((files) => emitted.push(files));

    const invalidFile = new File(['x'], 'invoice.pdf', { type: 'application/pdf' });
    component['emitValidatedFiles']([invalidFile]);

    expect(emitted).toHaveLength(0);
    expect(component.validationMessages()).toContain('"invoice.pdf" no es un tipo de archivo admitido.');
  });

  it('should emit files that pass validation', () => {
    const emitted: File[][] = [];
    component.filesSelected.subscribe((files) => emitted.push(files));

    const validFile = new File(['<xml/>'], 'invoice.xml', { type: 'application/xml' });
    component['emitValidatedFiles']([validFile]);

    expect(emitted).toHaveLength(1);
    expect(emitted[0][0].name).toBe('invoice.xml');
  });
});
